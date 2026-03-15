# Watchflare — Internal Components

Detailed breakdown of every package and module in the codebase.

---

## Backend

The backend is a single Go binary. `main()` starts an HTTP server (Gin) and a gRPC
server concurrently, then launches three background workers.

### main.go

Entry point. Responsibilities:
- Load configuration, connect to database, initialise PKI.
- Start the HTTP router (`setupRouter`) and the gRPC server (`startGRPCServer`).
- Launch three background goroutines:
  - **SyncWorker** — flushes heartbeat cache to DB every 5 min.
  - **StaleChecker** — scans cache every 10 s, marks agents offline after 15 s silence.
  - **AggregatedMetricsScheduler** — computes cross-server aggregates every 30 s and
    broadcasts them via SSE.

`setupRouter` registers all HTTP routes in two groups: public (`/auth/register`,
`/auth/login`, `/auth/logout`, `/health`) and protected (everything under `/servers`
and the remaining `/auth` endpoints). The protected group passes through
`middleware.AuthMiddleware()`.

---

### config/

**config.go** — Loads all settings from environment variables with sensible defaults.
Exposes a single `AppConfig` global used everywhere.

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | 8080 | HTTP listen port |
| `GRPC_PORT` | 50051 | gRPC listen port |
| `JWT_SECRET` | — (required) | HMAC key for JWT signing. Must be ≥ 32 chars |
| `CORS_ORIGINS` | `http://localhost:5173` | Comma-separated allowed origins |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `TLS_MODE` | `auto` | `auto` (generate certs) or `custom` |
| `GRPC_TIMESTAMP_WINDOW` | 300 | HMAC anti-replay window in seconds |
| `POSTGRES_*` | see db | Connection params for PostgreSQL |

Startup is fatal if `JWT_SECRET` is missing. A warning is logged if it contains
common weak substrings (`secret`, `password`, `admin`, `test`, …).

---

### database/

**db.go** — Manages the entire PostgreSQL lifecycle. On `Connect()`:

1. Opens a GORM connection to PostgreSQL.
2. Enables the `timescaledb` extension.
3. Auto-migrates `users` and `servers` tables via GORM.
4. Creates the `metrics` table manually (TimescaleDB compression requires explicit
   control — GORM's auto-migrate would conflict).
5. Converts `metrics` into a TimescaleDB **hypertable** partitioned on `timestamp`.
6. Enables compression on the hypertable.
7. Adds a **30-day retention policy** — older rows are automatically dropped.
8. Runs four embedded SQL migrations in order:
   - `001_continuous_aggregates.sql` — pre-computed aggregate views (10 min, 15 min,
     2 h, 8 h buckets) used by the per-server metrics endpoint.
   - `002_dropped_metrics.sql` — tracking table + summary view for metrics the agent
     could not deliver.
   - `003_packages.sql` — `packages`, `package_history`, `package_collections` tables.
   - `004_environment_detection.sql` — adds environment-type columns to `servers`.

All migration SQL is embedded at compile time (`//go:embed`). No external files are
needed at runtime.

---

### models/

GORM model definitions. Each has a `BeforeCreate` hook that generates a UUID if the
ID is empty.

**User** — `id`, `email` (unique), `password` (bcrypt hash, excluded from JSON via
`json:"-"`), `default_time_range`, `theme`, timestamps.

**Server** — the record for one monitored server. Grouped logically:
- *Admin fields:* `name`, `configured_ip`, `allow_any_ip_registration`,
  `ignore_ip_mismatch`.
- *Registration fields:* `registration_token` (SHA-256 hash), `expires_at`, `agent_id`,
  `agent_key`. Token and expiry are nulled after successful registration.
- *Agent-reported fields:* `hostname`, `ip_address_v4/v6`, `platform`,
  `platform_version`, `platform_family`, `architecture`, `kernel`,
  `environment_type`, `hypervisor`, `container_runtime`.
- *Status:* `status` (`pending` → `online` / `offline` / `expired`), `last_seen`.

**Metric** — one time-series sample. Maps directly to the TimescaleDB hypertable row.
Fields: `server_id`, `timestamp`, CPU, memory (total/used/available), load averages
(1/5/15 min), disk (total/used), uptime.

**Package / PackageHistory / PackageCollection** — current installed packages,
historical change log (with `change_type`: added / removed / updated / initial), and
metadata about each collection run (timestamp, type, duration, status).

---

### handlers/

HTTP handlers, one file per domain. Each handler binds the request, calls the
appropriate service or queries the database, and returns JSON.

**auth.go**
| Handler | Method | Route | Notes |
|---------|--------|-------|-------|
| SetupRequired | GET | `/auth/setup-required` | Public. Returns `{setup_required: bool}` |
| Register | POST | `/auth/register` | Creates first user only. Sets JWT cookie |
| Login | POST | `/auth/login` | Validates credentials. Sets JWT cookie |
| Logout | POST | `/auth/logout` | Clears JWT cookie (maxAge = -1) |
| GetCurrentUser | GET | `/auth/user` | Protected. Returns user + preferences |
| UpdatePreferences | PUT | `/auth/preferences` | Protected. Validates time_range and theme |
| ChangePassword | PUT | `/auth/change-password` | Protected. Requires current password |

JWT cookie: HttpOnly, SameSite=Lax, Secure in production, 7-day expiry.

**servers.go**
| Handler | Method | Route | Notes |
|---------|--------|-------|-------|
| CreateAgent | POST | `/servers` | Calls `services.CreateAgent`, returns token + key |
| ListServers | GET | `/servers` | |
| GetServer | GET | `/servers/:id` | |
| ValidateIP | PUT | `/servers/:id/validate-ip` | Sets detected IP, clears configured IP |
| UpdateConfiguredIP | PUT | `/servers/:id/change-ip` | Saves previous IP before overwriting |
| IgnoreIPMismatch | PUT | `/servers/:id/ignore-ip-mismatch` | |
| RegenerateToken | POST | `/servers/:id/regenerate-token` | Only for pending/expired servers |
| DeleteServer | DELETE | `/servers/:id` | |
| GetDroppedMetrics | GET | `/servers/dropped-metrics` | Queries `agent_dropped_metrics_summary` view |
| GetAggregatedMetrics | GET | `/servers/metrics/aggregated` | Time-bucketed cross-server aggregation |

`GetAggregatedMetrics` uses a two-level CTE: first averages per server per bucket
(avoids inflating values when a server reports multiple times), then sums/averages
across servers. Interval mapping:

| `time_range` | Duration | Bucket interval |
|--------------|----------|-----------------|
| `1h` | 1 hour | 30 s |
| `12h` | 12 hours | 10 min |
| `24h` | 24 hours | 20 min |
| `7d` | 7 days | 2 h |
| `30d` | 30 days | 8 h |

**metrics.go** — `GetMetrics`: per-server historical metrics. Accepts `time_range`
or explicit `start`/`end`/`interval`. Routes to raw data or continuous aggregate
views depending on the requested interval.

**packages.go** — Four endpoints for package data:
- `GetServerPackages` — paginated list with search (ILIKE) and package_manager filter.
- `GetServerPackageHistory` — change log, filterable by change_type and time range.
- `GetServerPackageCollections` — metadata about each collection run.
- `GetPackageStats` — summary: total packages, breakdown by manager, recent changes.

**sse.go** — `ServerEvents`: upgrades the connection to SSE. Sets appropriate headers
(`text/event-stream`, `Cache-Control: no-cache`), registers a client with the broker,
streams events until the client disconnects.

---

### middleware/

**jwt.go** — `AuthMiddleware()`. For every protected request:
1. Reads the `jwt_token` cookie.
2. Parses and verifies the JWT signature (HS256, key = `JWT_SECRET`).
3. Extracts `user_id` from claims.
4. Queries the database to confirm the user still exists. A deleted user's valid
   token is immediately rejected.
5. Stores `user_id` in the Gin context for downstream handlers.

---

### services/

Business logic, decoupled from HTTP concerns.

**auth_service.go** — `Register` (checks no users exist first), `Login`, `ChangePassword`,
`generateJWT`. Password hashing uses bcrypt at default cost (10).

**server_service.go** — `CreateAgent`: generates `agent_id` (UUID), `registration_token`
(`wf_reg_` + 32 hex chars from 16 random bytes), hashes the token, generates a 32-byte
`agent_key`, creates the Server record with status `pending` and 24 h expiry.
Also: `ValidateIP`, `UpdateConfiguredIP`, `RegenerateToken`, `DeleteServer`.

**metrics_service.go** — `GetMetrics` query logic. Without an interval: raw rows from
the `metrics` table. With an interval: routes to the appropriate continuous aggregate
view (`hourly_10m`, `hourly_15m`, `daily_2h`, `monthly_8h`).

**aggregated_metrics_scheduler.go** — Runs a ticker aligned to 30 s boundaries
(`time.Truncate`). Each tick queries the last 30 s of metrics across all online
servers, computes AVG(cpu) / SUM(memory, disk), and broadcasts an
`aggregated_metrics_update` event via SSE. The bucket timestamp is the truncated
time (e.g. 07:10:30), not the query time.

---

### grpc/

**agent_service.go** — Implements all five RPCs defined in `agent.proto`:

| RPC | Auth | What it does |
|-----|------|--------------|
| `RegisterServer` | Token (no HMAC) | Validates token, updates server record, returns credentials + CA cert |
| `Heartbeat` | HMAC | Updates in-memory cache, broadcasts SSE |
| `SendMetrics` | HMAC | Inserts into `metrics` table, broadcasts SSE |
| `ReportDroppedMetrics` | HMAC | Inserts into `dropped_metrics` table |
| `SendPackageInventory` | HMAC | Processes full or delta inventory in a DB transaction |

`SendPackageInventory` runs inside a transaction. For delta mode it processes three
lists (added → upsert + history "added", removed → delete + history "removed",
updated → update + history "updated"). A `package_collections` metadata row is
created on commit.

**interceptor.go** — Unary interceptor applied to every gRPC call. Skips
`RegisterServer`. For all others: extracts `x-watchflare-hmac` and
`x-watchflare-timestamp` from metadata, looks up the `agent_key` by `agent_id`,
validates the timestamp is within the configured window, recomputes the HMAC,
and compares with `hmac.Equal` (constant-time).

**validator.go** — Stateless helpers: `ValidateTimestamp`, `ValidateHMAC`,
`computeHMAC`. Uses reflection to extract `AgentId` and `Timestamp` fields from
any protobuf message, keeping the interceptor generic across all RPCs.

---

### pki/

Two modes controlled by `TLS_MODE`:

**auto** — On first startup, if `./pki/` does not contain certificates:
- Generate a self-signed CA (RSA-4096, valid 10 years).
- Generate a server certificate signed by the CA (valid 5 years, SANs: `watchflare`,
  `localhost`).
- Save all files with appropriate permissions (keys at 0600).

**custom** — Operator supplies `ca.pem`, `server.pem`, `server-key.pem`.
The module validates they exist and can be parsed.

`GetTLSConfig()` returns a `tls.Config` with both `MinVersion` and `MaxVersion`
set to TLS 1.3. No older protocol is negotiable.

`GetCACertPEM()` returns the CA certificate in PEM format — this is what gets
sent to agents during registration.

---

### cache/

**heartbeat.go** — Thread-safe (`sync.RWMutex`) map keyed by `agent_id`. Each entry
stores `LastSeen`, `Status`, `IPv4`, `IPv6`, and an `Updated` flag. `Update()`
sets the flag; `MarkSynced()` clears it after the DB write.

**sync_worker.go** — Two goroutines sharing the same cache:
- **SyncWorker**: ticks every 5 min. Iterates the cache, writes only entries where
  `Updated == true` to the `servers` table, then marks them synced.
- **StaleChecker**: ticks every 10 s. Calls `cache.CheckStale(threshold)` which
  compares `LastSeen` against the threshold (15 s). Any agent past the threshold
  is marked offline in the cache and an SSE `server_update` event is broadcast.

---

### sse/

**broker.go** — Singleton broker managing all SSE connections.

Each client gets a buffered channel (size 10). `Broadcast` sends to every client;
if a channel is full the event is dropped for that client (logged as warning).

Four event types:
| Event | Payload | Trigger |
|-------|---------|---------|
| `server_update` | status, IPs, last_seen | Heartbeat, registration, stale detection |
| `metrics_update` | minified single-server metrics | SendMetrics RPC |
| `aggregated_metrics_update` | cross-server aggregates | AggregatedMetricsScheduler |
| `connected` | client_id | On SSE connection established |

**Minification:** `metrics_update` payloads compress field names to single letters
(`s`=server_id, `t`=timestamp, `c`=cpu, `mu`=mem_used, …) and convert timestamps
to Unix epoch integers. The frontend decodes these back to full names.

---
---

## Agent

The agent is a single Go binary installed on each monitored server. It stores
configuration in `/etc/watchflare/` and runtime data (WAL) in `/var/lib/watchflare/`.
Both paths are overridable via environment variables.

### main.go

Two execution modes selected by the first CLI argument:

**`register`** — One-time setup:
1. Collect system info (hostname, IPs, platform, kernel, environment type).
2. Connect to backend with permissive TLS (no CA cert yet).
3. Send `RegisterServer` RPC with the operator-provided token.
4. Receive `agent_id`, `agent_key`, CA cert.
5. Save CA cert to `/etc/watchflare/ca.pem`.
6. Save all credentials to `/etc/watchflare/agent.conf`.

**Normal mode** (no argument):
1. Load config from disk, validate required fields.
2. Create gRPC client with strict TLS (CA cert verification).
3. Initialise WAL.
4. Detect environment type, derive `MetricsConfig`.
5. Create `Sender` (metrics collector + WAL + gRPC client).
6. Spawn three goroutines: heartbeat, sender, package collector.
7. Block on `SIGINT` / `SIGTERM`. On signal: cancel context, wait briefly for
   sender's graceful flush, exit.

---

### config/

**config.go** — TOML configuration file. Loaded from `{ConfigDir}/agent.conf`.

| Field | Default | Purpose |
|-------|---------|---------|
| `server_host` | — | Backend hostname |
| `server_port` | — | Backend gRPC port |
| `agent_id` | — | UUID assigned during registration |
| `agent_key` | — | HMAC signing key (hex) |
| `ca_cert_file` | — | Path to CA cert (set during registration) |
| `heartbeat_interval` | 5 | Seconds between heartbeats |
| `metrics_interval` | 30 | Seconds between metric collections |
| `wal_enabled` | — | Enable/disable WAL |
| `wal_path` | `{DataDir}/metrics.wal` | WAL file location |
| `wal_max_size_mb` | 10 | MB before FIFO truncation |

File permissions: config directory 0750, config file 0640.

---

### client/

**grpc.go** — gRPC client wrapper. Two constructors:
- `New()` — strict TLS. Loads CA cert, sets `ServerName` for SNI verification.
- `NewForRegistration()` — `InsecureSkipVerify: true`. Used only during the
  registration RPC.

Every post-registration RPC method:
1. Sets a context timeout (5 s default; 30 s for package inventory).
2. Calls `security.AttachAuthMetadata()` to compute and attach the HMAC.
3. Invokes the protobuf-generated client method.

---

### security/

**hmac.go** — Two functions:

`ComputeHMAC(agentKey, timestamp, agentID, protoMessage)` builds the payload:
```
[8-byte big-endian timestamp] + "|" + agentID + "|" + marshal(protoMessage)
```
Signs it with HMAC-SHA256, returns the hex-encoded signature.

`AttachAuthMetadata(ctx, …)` computes the HMAC and attaches two gRPC metadata
headers: `x-watchflare-hmac` (signature) and `x-watchflare-timestamp` (epoch string).

The 8-byte binary timestamp format avoids any string-parsing ambiguity or padding
attack that a decimal string representation might allow.

---

### metrics/

**collector.go** — Collects system metrics using `gopsutil`:

| Metric | Source | Skipped when |
|--------|--------|--------------|
| CPU usage % | 1 s sample | — |
| Memory total / used / available | `/proc/meminfo` | — |
| Load average 1 / 5 / 15 min | `/proc/loadavg` | — |
| Disk total / used | root partition | `CollectDisk == false` (containers) |
| Uptime | `/proc/uptime` | — |

The `MetricsConfig` (from `sysinfo`) controls which metrics are actually collected.
Errors on individual metrics are silently swallowed — the field stays at zero.

---

### sysinfo/

**environment.go** — Detects the runtime environment at startup. Detection order:

1. **Container?** Check `/.dockerenv`, scan `/proc/1/cgroup` for docker/lxc/kubepods/
   podman, inspect `/proc/1/cmdline` to see if PID 1 is not init/systemd.
2. **VM?** Read `/sys/class/dmi/id/product_name` and `sys_vendor` for hypervisor
   keywords (vmware, virtualbox, kvm, qemu, microsoft, xen, …).
3. **Docker running on host?** Check existence of `/var/run/docker.sock`.
4. **Hypervisor / runtime identity** — the specific hypervisor or container runtime
   string (e.g. `kvm`, `docker`, `kubernetes`).

Result is an `Environment` struct sent to the backend during registration and used
locally to derive `MetricsConfig`.

**metrics_config.go** — Maps `EnvironmentType` → `MetricsConfig`:

| Environment | Collects disk | Collects swap | Collects temperature |
|-------------|---------------|---------------|----------------------|
| Physical | yes | yes | yes |
| Physical + Docker | yes | yes | yes |
| VM | yes | no | no |
| VM + Docker | yes | no | no |
| Container | **no** | no | no |

Container disk is skipped because disk is shared with the host — reporting it from
every container would inflate the dashboard totals.

---

### wal/

Write-Ahead Log. Guarantees metric durability across crashes and network outages.

**wal.go** — File-based append-only log.

Record format on disk:
```
┌──────────┬──────────────┬─────────┐
│ 4 bytes  │  N bytes     │ 4 bytes │
│ length   │  payload     │ CRC32   │
│ (uint32, │  (protobuf   │ (IEEE,  │
│  big-end)│   bytes)     │  big-end│
└──────────┴──────────────┴─────────┘
```

Key operations:
- **Append** — writes one record, calls `file.Sync()` for durability.
- **ReadAll** — reads from the start, validates each CRC32, returns all payloads.
- **Clear** — truncates the file to zero bytes.
- **Truncate (FIFO)** — when the file exceeds `maxSize`, keeps the most recent 50 %
  of records. Crash-safe sequence:
  1. Read all current records.
  2. Write the kept records to a temp file, sync it.
  3. Sync the directory (ensures the temp file's directory entry is durable).
  4. Atomic rename temp → WAL path.
  5. Reopen the WAL file.

  If the process crashes at any point, either the old file or the new file is intact.

On startup, any leftover temp file from a previous crash is deleted.

**sender.go** — Orchestrates the metrics pipeline every `MetricsInterval` seconds:

```
1. Collect fresh metrics          (metrics.Collect)
2. Serialize to protobuf bytes    (serializeMetrics)
3. Append to WAL                  (wal.Append)
4. If WAL > maxSize → Truncate    (wal.Truncate, FIFO)
5. Read all WAL records           (wal.ReadAll)
6. Send each record via gRPC      (client.SendMetrics)
7. If ALL succeed → Clear WAL     (wal.Clear)
   If ANY fail   → keep everything for next cycle
```

**Startup replay:** If records exist in the WAL from a previous run (crash or
incomplete shutdown), they are sent before any new metrics are collected.

**Graceful shutdown:** On context cancellation, `shutdown()` attempts a final flush
with a 5 s timeout. If it succeeds, the WAL is cleared. If it times out, the WAL
is left intact for replay on next startup.

---

### packages/

**registry.go** — Central registry of all package collectors. `NewRegistry()` registers
collectors based on `runtime.GOOS`:
- **Darwin:** brew, macports, npm, pip, gem, cargo, composer, yarn, pnpm, poetry,
  pipx, uv, conda, mamba, nuget, maven, nix, cli_tools.
- **Linux:** dpkg, rpm, pacman, apk, zypper, snap, flatpak, appimage, npm, pip, gem,
  cargo, composer, yarn, pnpm, poetry, pipx, uv, conda, mamba, nuget, maven, nix,
  cli_tools.

`GetAvailableCollectors()` filters to only those where `IsAvailable()` returns true
(the relevant CLI tool is installed and in PATH).

**Collector interface:**
```go
type Collector interface {
    Name() string
    IsAvailable() bool
    Collect() ([]*Package, error)
}
```

Each concrete collector runs the package manager CLI, parses stdout (JSON or text
depending on the tool), and returns a slice of `Package` structs.

**state.go** — Persists the last known package state as JSON. `ComputeDelta(old, new)`
diffs by a composite key `name|package_manager` and returns three slices: added,
removed, updated.

**cli_tools.go** — A special collector that scans ~40 common CLI tools (docker,
kubectl, git, python, go, aws, terraform, …). For each: checks if it is in PATH,
runs a version command (with regex to extract the version number), and returns it
as a package entry with `package_manager = "cli-tools"`.

---
---

## Frontend

SvelteKit 5 application with Tailwind CSS. All API communication goes through
`lib/api.js`. Real-time updates come via `lib/sse.js`.

### Routes

| Route | Page | Description |
|-------|------|-------------|
| `/` | `+page.svelte` | Dashboard. Stat cards (servers, CPU, RAM, disk). CPU / Memory / Disk charts. Dropped-metrics alert banner. Time-range selector. |
| `/servers` | `servers/+page.svelte` | Server list. Real-time status badges. IP mismatch warnings. Delete button. |
| `/servers/new` | `servers/new/+page.svelte` | Add-server form (name, IP, allow-any-IP). On success: displays token, agent key, and install instructions. |
| `/servers/[id]` | `servers/[id]/+page.svelte` | Server detail. System info grid (platform, arch, environment, hypervisor). IP mismatch card with action buttons. Token regeneration. Package stats summary. Delete modal. |
| `/servers/[id]/packages` | `servers/[id]/packages/+page.svelte` | Package list: search, filter by manager, pagination (50/page). Collection history toggle. Per-manager stats grid. |
| `/login` | `login/+page.svelte` | Email + password login form. |
| `/register` | `register/+page.svelte` | First-user registration form. |
| `/settings` | `settings/+page.svelte` | Password change form (current + new + confirm). |

### lib/

**api.js** — Fetch wrapper used by every page.
- All requests include `credentials: 'include'` so the browser sends the JWT cookie.
- On a 401 response: calls `checkSetupRequired()`. If no users exist, redirects to
  `/register`; otherwise redirects to `/login`.
- Exports one function per API operation (grouped by domain: auth, servers, metrics,
  packages).

**sse.js** — `connectSSE(onMessage, onError)`:
- Opens an `EventSource` to `/servers/events` with `withCredentials: true`.
- Listens for four named events: `connected`, `server_update`, `metrics_update`,
  `aggregated_metrics_update`.
- Decodes minified `metrics_update` payloads back to full field names and converts
  Unix timestamps to ISO strings before passing to the callback.
- Returns a disconnect function (calls `eventSource.close()`).

**utils.js** — `formatBytes(n)` (human-readable byte sizes), `formatPercent(n)`,
and `cn()` (Tailwind class merging via clsx + tailwind-merge).

### Components

| Component | Purpose |
|-----------|---------|
| `StatCard` | Metric summary card: title, value, subtitle, optional percentage bar, status badge |
| `TimeRangeSelector` | Button group: 1h / 12h / 24h / 7d / 30d. Emits value change |
| `CPUChart` | Line chart of CPU usage over time |
| `MemoryChart` | Stacked area chart of memory used vs total |
| `DiskChart` | Stacked area chart of disk used vs total |
| `InstallInstructions` | Renders the `curl` install command with copy-to-clipboard |

### Real-time data flow in the dashboard

1. On mount: load servers, dropped-metrics alerts, and aggregated metrics (full
   history for the selected time range) via REST.
2. Connect SSE.
3. `server_update` events update the server list reactively (status, IPs).
4. `aggregated_metrics_update` events append new 30 s data points to the chart arrays
   (capped at 200 points). This gives live chart updates on the 1 h view. Other time
   ranges receive their full dataset on load; new 30 s points are appended but are
   visually imperceptible at coarser scales.
5. On time-range change: reload the full aggregated-metrics history from the API.
