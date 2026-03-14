# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Watchflare is a self-hosted server monitoring platform with three components:
- **Backend** (Go): Central server with gRPC (agents) and HTTP (web dashboard) endpoints
- **Agent** (Go): Lightweight binary deployed on monitored servers
- **Frontend** (SvelteKit 5): Real-time dashboard with SSE

**Data flow**: Agents → gRPC/TLS 1.3 → Backend → PostgreSQL/TimescaleDB → SSE → Frontend

---

## Versioning

**Current version**: 0.26.2

**Versioning scheme** (pre-v1.0):
- `0.x.0` - Major features, redesigns, significant architectural changes
- `0.x.x` - Bug fixes, minor improvements, small features

**Rule**: Every commit MUST bump the version (major, minor, or patch). Update `frontend/package.json` and this file accordingly.

**v1.0 criteria** (not yet reached):
- All planned features implemented
- UI/UX polished and finalized
- Zero known bugs
- Production-ready quality
- User validation complete

**Version history:**
- `0.26.2` - Add container metrics continuous aggregates (10min, 15min, 2h, 8h buckets) matching system metrics pattern, backend queries aggregated views for 12h/24h/7d/30d (~90 points vs ~11600 raw), remove debug logs
- `0.26.1` - Fix charts invisible when browser/server clocks differ (anchor x-axis to max of browser and data timestamps), cache resolveColor() across chart instances, optimize time range switching (setData instead of destroy/recreate, dynamic gap threshold, tick interval in $effect)
- `0.26.0` - Chart polish: wall-clock x-axis (anchored to Date.now, auto-tick per time range), native gap detection via series.gaps (no synthetic nulls), isolated point dots, custom cursor overlay (dashed line + hover dots aligned via valToPos), area fill on all 10 charts, date labels for 7d/30d views, cleanup unused hasData variables
- `0.25.0` - Migrate charts from layerchart (SVG) to uPlot (Canvas): fixes Firefox OOM on server detail page, new UPlotChart wrapper with custom tooltip plugin, touch support (horizontal swipe), theme-reactive color resolution (oklch→hex for Canvas 2D), responsive resize via window event, cursor snap to nearest data point, clean Y-axis ticks (5 evenly spaced nice values), HH:MM X-axis format, nice rounding for byte/rate scales, removed layerchart/d3-scale dependencies
- `0.24.0` - Docker per-container metrics: agent collects CPU/memory/network per container via Docker API unix socket, new ContainerMetric protobuf message, backend stores in container_metrics hypertable (migration 007), SSE container_metrics_update event, 3 new chart components (ContainerCPU, ContainerMemory, ContainerNetwork) with dynamic series per container, automatic display on servers with containers
- `0.23.0` - New server metrics full-stack: agent collects disk I/O, network bandwidth, CPU temperature via delta tracking, protobuf fields 12-16, backend migration 006 (new columns + recreated continuous aggregates), SSE minified fields (dr/dw/nr/nt/tmp), 4 new chart components (LoadAvg, DiskIO, Network, Temperature), custom tooltip with layerchart primitive, yAxis formatting (%, bytes/sec), temperature conditional on physical servers, memory calculation changed to Total-Available
- `0.22.0` - Docker production deployment: single binary with embedded frontend (go:embed + build tags), multi-stage Dockerfile, docker-compose.prod.yml, SPA mode (adapter-static), API routes prefixed with /api, README and installation guide
- `0.21.0` - Pause/resume server monitoring (server-side): backend ignores heartbeats/metrics for paused servers, stale checker skips them, aggregated scheduler skips empty broadcasts, offline badge now red pastel, paused badge grey, pause/resume button on server detail page, status filter option
- `0.20.1` - Fix chart overflow when data has gaps (sleep/wake), filter data points to xDomain window
- `0.20.0` - Fix macOS APFS disk metrics (platform-specific diskutil collection), unified chart architecture (shared xDomain/tooltip/formatting in chart-utils.ts + ChartTooltip component), xDomain anchored to last data point for time range coverage, responsive chart headers (% only on mobile), tooltip color bar fix, formatBytes switched to base 1024 (IEC)
- `0.19.2` - Replace exact metric values with colored progress bars in server table, neutral text for percentages
- `0.19.1` - Full user settings page: email change, password change (migrated from /settings), preferences (theme + time range), standalone themeStore for cross-component reactivity, layout-level user loading with ready gate, /settings placeholder
- `0.19.0` - User menu in sidebar: dropdown with avatar/email replacing logout button, theme switcher (light/dark/system) with backend persistence, dropdown-menu UI components (bits-ui), placeholder user settings page
- `0.18.5` - Persist Global Metrics collapse state in localStorage across page refresh and reconnection
- `0.18.4` - UI polish: sort buttons as rounded hover pills on table headers (h-8, hover:bg-muted), default sort order changed to descending, removed environment filter from server list
- `0.18.3` - Rename server: new PUT /servers/:id/rename endpoint, rename modal in server detail page with pre-filled current name, 2-64 char validation
- `0.18.2` - Token regeneration: restricted to pending servers only (backend + frontend), replaced token modal with InstallInstructions component showing curl commands, added warning banner with copy token button, 24h expiry notice
- `0.18.1` - Collapsible Global Metrics: chevron toggle on dashboard, cards animate to compact mode (no icon/trend, smaller text), charts slide in/out with 250ms transitions, state persisted in UI store
- `0.18.0` - P5 polish: extracted buildQueryString() utility (6 API functions refactored), normalized handler names (handle* convention), added 24 new tests (toasts store, metrics store, buildQueryString) bringing total to 111
- `0.17.1` - P4 accessibility: added scope="col" to all 21 table headers (3 files), aria-invalid/aria-describedby on 10 form fields (4 files), RightSidebar close button already compliant
- `0.17.0` - P3 stores & performance: extracted 12 named constants replacing magic numbers, memoized dashboardStats derived store (skips recalculation on irrelevant SSE heartbeats), normalized store APIs with resetSidebar() and documented store categories in index.ts
- `0.16.0` - P2 component architecture: eliminated props drilling (alertCount store, authActions store, RightSidebar uses servers store directly), extracted 5 server sub-components (ServerDetailHeader, ServerAlerts, ServerMetricsCharts, ServerFilters, ServerListTable), server detail page 630→280 lines, server list page 550→200 lines
- `0.15.0` - P1 deduplication: extracted Modal, ConfirmDialog, Pagination reusable components, refactored 5 modals and 2 paginations, shared handleSSEReactivation utility across 3 pages, removed unused toasts imports
- `0.14.0` - P0 cleanup: all components migrated to Svelte 5 runes ($props, $state, $derived), TypeScript annotations on all .svelte files, InstallInstructions rewritten with Tailwind design tokens replacing 30 hardcoded colors, fixed PackageStats type to match backend
- `0.13.0` - Redesign header: command palette search (Cmd+K), Add Server button, always-visible alerts, centered W logo on mobile linking to dashboard
- `0.12.0` - Redesign server detail page: compact header card with inline info grid, live metrics in chart headers, packages section promoted with dedicated link
- `0.11.0` - Custom Select components (bits-ui) replacing native selects, dynamic width, renamed sidebar token to surface
- `0.10.1` - Harmonized right sidebar with floating card design (rounded-2xl, border, margin), wrapped servers page in card
- `0.10.0` - Shared layout: extracted sidebar/header/main wrapper into (app) route group layout, cleaned 6 pages from duplicated boilerplate
- `0.9.1` - Sidebar collapse UX: smooth width transition with text opacity animation, centered square icon backgrounds when collapsed (46x46px), Lucide nav icons, centered logo/SSE dot/logout when collapsed
- `0.9.0` - Frontend optimization: configurable API URL, deduplicated utils (getStatusClass, formatRelativeTime, countAlerts, generateAlerts), replaced any types, CSS design tokens, fixed Svelte 5 anti-patterns, dev-only logger, Escape key on modals, Lucide icons replacing {@html}, removed unused svelte.config alias
- `0.8.0` - Responsive redesign: harmonized breakpoints (sm/md/lg/xl), overlay alerts panel with bell badge, smooth sidebar collapse transition with text clipping, backend server sort/filter/search, mobile-first layouts on all pages
- `0.7.2` - Expand API tests with fetch mocking: login, register, CRUD servers, changePassword, metrics (82 tests)
- `0.7.1` - Frontend unit tests with Vitest: validation schemas, utility functions, API error handling (67 tests)
- `0.7.0` - Zod form validation (login, register, add server, change password), server list pagination (20/page), dashboard lazy loading (SSE-only for individual metrics), fix 401 redirect loop on auth pages
- `0.6.2` - Fix aggregated charts: bucket labels now represent end time (08:40 = avg 08:30-08:40), exclude incomplete buckets, fill CA materialization gap with raw metrics, auto-reload on bucket completion
- `0.6.1` - Fix SSE metrics polluting charts on 12h/24h/7d/30d views by snapping to correct time buckets
- `0.6.0` - Database optimization: global metrics endpoint now uses continuous aggregates (metrics_10min/15min/2h/8h) for 12h/24h/7d/30d time ranges, cross-server aggregation with JOIN + GROUP BY, raw metrics kept for 1h view
- `0.5.0` - SSE optimization: centralized SSE manager with auto-reconnection, connection pooling across pages, fixed aggregated metrics scheduler (2s latency vs 27s), real-time chart updates with {#key} reactivity, SSE status indicator in sidebar
- `0.4.0` - Centralized state management: 7 new stores (user, servers, metrics, aggregated, alerts, ui), refactored +page.svelte (-120 lines), fixed TimeRangeSelector reactivity
- `0.3.0` - Complete TypeScript migration: converted all .js to .ts, centralized types, improved error handling with ApiError class, refactored dashboard components
- `0.2.1` - Responsive improvements: fixed header, separate mobile/desktop sidebars with smooth animations
- `0.2.0` - Dashboard layout improvements: collapsible sidebars, stats cards with trends, right panel with alerts
- `0.1.1` - Fix macOS CPU metrics always showing 0% (gopsutil initialization bug)
- `0.1.0` - Initial frontend redesign with military green theme, sidebar navigation, table layouts

**Commit messages:**
- First line: Short summary (required)
- Optional: Up to 4 additional lines with brief details prefixed by "-" (not too detailed)
- Put comprehensive changes in this file's version history, not in commit messages
- Example format:
  ```
  Redesign frontend with minimal military green theme (v0.1.0)

  - New components: Sidebar, ServerTable, CompactStats
  - Table layouts replacing grids throughout
  - Reduced complexity in server details and packages pages
  ```

**Version updates:**
- Always update `frontend/package.json` version when improving the frontend
- Update version in this file's version history with brief description

---

## Build & Run Commands

**Rule**: Always use named output binaries when building Go projects: `go build -o watchflare-backend` (backend) and `go build -o watchflare-agent` (agent). Never use bare `go build` or `go build ./...` for production builds.

### Backend
```bash
cd backend

# Development (auto-reload requires external tool)
go run .

# Build
go build -o watchflare-backend

# Run tests
go test ./...                           # All tests
go test ./handlers -v                   # Specific package
go test -run TestCreateAgent ./services # Single test
```

**Environment**: Copy `.env.example` to `.env` and configure. Required: `JWT_SECRET` (≥32 chars).

**Test credentials**: `admin@watchflare.io` / `watchflare_p4ss`

### Agent
```bash
cd agent

# Build (current platform)
go build -o watchflare-agent

# Build all architectures
./build-all.sh
# Outputs to ./dist/:
#   - linux_amd64/watchflare-agent
#   - linux_arm64/watchflare-agent
#   - darwin_amd64/watchflare-agent
#   - darwin_arm64/watchflare-agent
#   - watchflare_checksums.txt

# Development run (requires existing config)
./watchflare-agent

# Register with backend
./watchflare-agent register --token=wf_reg_... --host=localhost --port=50051

# Install as system service (macOS)
# Option A: Install + register + start in one command
sudo ./install-macos.sh --token=wf_reg_... --host=localhost --port=50051

# Option B: Install only (register manually after)
sudo ./install-macos.sh

# Install as system service (Linux)
# Option A: Install + register + start in one command
sudo ./install-linux.sh --token=wf_reg_... --host=localhost --port=50051

# Option B: Install only (register manually after)
sudo ./install-linux.sh

# Run tests
go test ./...
go test ./wal -v                        # WAL tests
go test ./packages -run TestRegistry    # Package collector tests
```

**Service management (macOS)**:
```bash
sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist  # Start
sudo launchctl bootout system/io.watchflare.agent                                 # Stop
tail -f /var/log/watchflare-agent.log                                              # Logs
```

**Service management (Linux)**:
```bash
sudo systemctl start watchflare-agent    # Start
sudo systemctl stop watchflare-agent     # Stop
sudo systemctl restart watchflare-agent  # Restart
sudo systemctl status watchflare-agent   # Status
journalctl -u watchflare-agent -f        # Logs (systemd journal)
tail -f /var/log/watchflare-agent.log    # Logs (file)
```

### Frontend
```bash
cd frontend

npm install          # First time only
npm run dev          # Development server (http://localhost:5173)
npm run build        # Production build
npm run preview      # Preview production build
```

### Database
```bash
docker compose up -d              # Start TimescaleDB
docker compose down               # Stop
docker compose logs -f postgres   # View logs

# Connect to DB
docker exec -it watchflare-postgres psql -U watchflare -d watchflare
```

**Default connection**: `postgresql://watchflare:watchflare_dev@localhost:5433/watchflare`

### Protobuf (rare)
```bash
cd shared/proto
protoc --go_out=. --go-grpc_out=. agent.proto
```

---

## Architecture Patterns

### Security Model

**Three authentication layers:**
1. **Agent registration** (one-time): Token-based (`wf_reg_...`), SHA-256 hashed in DB, 24h expiry
2. **Agent RPCs** (ongoing): HMAC-SHA256 per request with timestamp anti-replay (±5min window)
3. **Web users**: JWT in HttpOnly cookie (7 day expiry), validated against DB on every request

**TLS 1.3 mandatory**: Auto-generated PKI in `./pki/` (CA + server cert) or custom certs. Agents pin CA cert during registration.

**Critical**: `agent_key` stored plaintext in DB (required for HMAC validation). Database access = ability to impersonate agents.

### WAL (Write-Ahead Log)

**Purpose**: Durability for metrics when backend is unreachable. **Enabled by default**.

**Design**: Single file (`/var/lib/watchflare/metrics.wal`), append-only with `fsync()` after each write.
- Format: `[Length:4 bytes][Protobuf data][CRC32:4 bytes]`
- FIFO truncation: Keeps 50% most recent when exceeds `wal_max_size_mb` (default 10MB)
- Atomic truncation: Temp file → sync → atomic rename (crash-safe)

**Flow**: Collect → Append to WAL → Send → Clear WAL only if all sends succeed.

**Recovery logging**: Explicit messages when replaying metrics after backend downtime:
- `"WAL RECOVERY: Found X pending metrics from previous backend downtime"`
- `"WAL cleared (recovery finished)"` or `"Sent X metrics (including Y accumulated during backend outage)"`

**Code**: `agent/wal/wal.go` (file operations), `agent/wal/sender.go` (orchestration).

### Package Inventory

**Registry pattern** (`agent/packages/registry.go`):
- 28 collectors registered at startup, filtered by OS and CLI availability
- Each collector implements: `Name()`, `IsAvailable()`, `Collect() ([]*Package, error)`

**Delta detection** (`agent/packages/state.go`):
- State saved to `/var/lib/watchflare/packages.state.json`
- First run: sends full inventory (`inventory_type: "full"`)
- Subsequent: sends only changes (`inventory_type: "delta"` with `added`, `removed`, `updated` lists)

**Collection schedule**:
- Initial: 60s after agent startup
- Recurring: Daily at 3:00 AM

**Backend processing** (`backend/grpc/agent_service.go`):
- Full mode: upserts all packages + creates initial history records
- Delta mode: processes changes in a single transaction
- Three tables: `packages` (current state), `package_history` (changelog), `package_collections` (metadata)

### Heartbeat & Cache Layer

**Three separate loops** (backend):
1. **Heartbeat ingestion** (5s): Updates in-memory cache, broadcasts SSE. No DB write.
2. **Stale checker** (10s): Scans cache, marks agents offline after 15s silence.
3. **Sync worker** (5min): Flushes cache changes to DB (`last_seen`, `status`, IPs).

**Code**: `backend/cache/heartbeat.go` (cache), `backend/cache/sync_worker.go` (workers).

**Why**: Decouples network rate (5s) from DB write rate (5min). Heartbeats are cheap.

### TimescaleDB Hypertables

**Automatic partitioning** on `timestamp` with compression and 30-day retention.

**Continuous aggregates** (precomputed views):
- `hourly_10m` / `hourly_15m`: 10/15 minute buckets for 1h/12h/24h views
- `daily_2h`: 2 hour buckets for 7-day view
- `monthly_8h`: 8 hour buckets for 30-day view

**Migrations** (`backend/database/migrations/`):
- `001`: Continuous aggregates
- `002`: Dropped metrics tracking
- `003`: Package tables
- `004`: Environment detection columns

All embedded at compile time (`//go:embed`).

### SSE (Server-Sent Events)

**Four event types** (`backend/sse/broker.go`):
- `server_update`: Status, IPs, last_seen (from heartbeat/stale detection)
- `metrics_update`: Single-server metrics (minified: `s`, `t`, `c`, `mu`, etc.)
- `aggregated_metrics_update`: Cross-server aggregates every 30s
- `connected`: Client connection confirmation

**Minification**: Field names compressed to 1-2 chars, timestamps as Unix epoch integers. Frontend decodes in `lib/sse.js`.

**Backpressure**: Buffered channel (size 10) per client. Full channel = event dropped + warning logged.

### Environment Detection

**Hierarchy** (`agent/sysinfo/environment.go`):
1. Container? (check `/.dockerenv`, `/proc/1/cgroup`, PID 1 process)
2. VM? (DMI product name for hypervisor keywords)
3. Docker on host? (`/var/run/docker.sock`)

**Result** (`EnvironmentType`): `physical`, `physical_with_containers`, `vm`, `vm_with_containers`, `container`

**Impact** (`agent/sysinfo/metrics_config.go`):
- Containers: Skip disk metrics (shared with host, would double-count)
- VMs: Skip swap and temperature (no physical access)

---

## Critical Entry Points

**Backend lifecycle**:
- `backend/main.go:main()` - Application bootstrap, starts HTTP + gRPC servers + 3 background workers
- `backend/database/db.go:Connect()` - DB connection, TimescaleDB setup, runs embedded migrations

**Agent lifecycle**:
- `agent/main.go:main()` - Mode switch (`register` vs normal operation)
- `agent/wal/sender.go:Run()` - Metrics collection loop (blocks until SIGINT/SIGTERM)

**gRPC handlers** (see `backend/grpc/agent_service.go`):
- `RegisterServer()` - Token validation, credential issuance, CA cert distribution
- `Heartbeat()` - Cache update, SSE broadcast (no DB write)
- `SendMetrics()` - Metrics ingestion, SSE broadcast
- `SendPackageInventory()` - Full/delta processing in single transaction

**HTTP handlers** (see `backend/handlers/`):
- `handlers/auth.go:Login()` - JWT issuance
- `handlers/auth.go:AuthMiddleware()` - JWT validation on every protected request
- `handlers/servers.go:CreateAgent()` - Token generation (SHA-256 hash stored)
- `handlers/sse.go:ServerEvents()` - SSE connection upgrade, client registration

**Package collection**:
- `agent/packages/registry.go:NewRegistry()` - Collector initialization (OS-specific + cross-platform)
- `agent/packages/registry.go:GetAvailableCollectors()` - Filters by `IsAvailable()`
- `agent/packages/state.go:ComputeDelta()` - Diff algorithm (composite key: `name|package_manager`)
- `agent/main.go:runPackageCollector()` - Scheduler (60s initial + daily 3 AM)

**Background workers** (launched in `backend/main.go`):
- `cache/sync_worker.go:SyncWorker()` - Heartbeat cache → DB flush (5 min)
- `cache/sync_worker.go:StaleChecker()` - Offline detection (10s scan, 15s threshold)
- `services/aggregated_metrics_scheduler.go:Start()` - Cross-server metrics (30s)

*For complete API surface, see `docs/internals.md`*

---

## Key File Locations

### Agent (when installed as service)

**macOS:**
```
/usr/local/bin/watchflare-agent       # Binary (root:wheel, 755)
/etc/watchflare/agent.conf            # Config with credentials (root:staff, 640)
/etc/watchflare/ca.pem                # Pinned CA cert (root:staff, 644)
/var/lib/watchflare/                  # Data directory (watchflare:staff, 750)
  ├── metrics.wal                     # WAL file
  └── packages.state.json             # Package inventory state
/var/log/watchflare-agent.log         # Logs (watchflare:staff, 644)
/Library/LaunchDaemons/io.watchflare.agent.plist  # Service definition
```

**Linux:**
```
/usr/local/bin/watchflare-agent       # Binary (root:root, 755)
/etc/watchflare/agent.conf            # Config with credentials (root:watchflare, 640)
/etc/watchflare/ca.pem                # Pinned CA cert (root:watchflare, 644)
/var/lib/watchflare/                  # Data directory (watchflare:watchflare, 750)
  ├── wal/metrics.wal                 # WAL file
  └── packages.state.json             # Package inventory state
/var/log/watchflare-agent.log         # Logs (watchflare:watchflare, 644)
/etc/systemd/system/watchflare-agent.service  # Systemd service definition
```

### Backend
```
backend/
  ├── .env                            # Configuration (git-ignored)
  ├── data/pki/                       # Auto-generated TLS certs (dev) [git-ignored]
  │   ├── ca.pem / ca-key.pem         # CA (10 year validity)
  │   └── server.pem / server-key.pem # Server cert (5 year validity)
  │                                   # Production: /var/lib/watchflare/pki/
  ├── database/migrations/            # SQL migrations (embedded)
  ├── grpc/agent_service.go           # 5 RPC handlers
  ├── handlers/                       # HTTP handlers (auth, servers, metrics, packages, sse)
  └── services/                       # Business logic
```

### Shared
```
shared/proto/agent.proto              # gRPC service definition
  ├── agent.pb.go                     # Generated protobuf code
  └── agent_grpc.pb.go                # Generated gRPC stubs
```

---

## Common Patterns & Gotchas

### Adding a New Package Collector

1. Create `agent/packages/{name}.go` implementing `Collector` interface
2. Add to registry in `agent/packages/registry.go`:
   - Platform-specific: `registerPlatformCollectors()` with OS switch
   - Cross-platform: `registerLanguageCollectors()`
3. Implement `IsAvailable()` to check if CLI exists: `exec.LookPath("binary-name")`
4. Parse CLI output in `Collect()`, return `[]*Package` with `package_manager` set to collector name
5. Test: `go test ./packages -run TestRegistry -v`

**Example structure**:
```go
type FooCollector struct{}
func (f *FooCollector) Name() string { return "foo" }
func (f *FooCollector) IsAvailable() bool { _, err := exec.LookPath("foo"); return err == nil }
func (f *FooCollector) Collect() ([]*Package, error) { /* ... */ }
```

### HMAC Interceptor

**All RPCs except `RegisterServer` require HMAC authentication.**

Agent-side (`agent/security/hmac.go`):
- `AttachAuthMetadata()` computes HMAC and adds `x-watchflare-hmac` + `x-watchflare-timestamp` headers
- Payload: `[8-byte timestamp]["|"][agent_id]["|"][marshaled protobuf]`

Backend-side (`backend/grpc/interceptor.go`):
- Uses reflection to extract `agent_id` and `timestamp` from any protobuf message
- Looks up `agent_key` from DB, recomputes HMAC, constant-time comparison

**When adding new RPCs**: Ensure protobuf message has `agent_id`, `agent_key`, and `timestamp` fields.

### Database Migrations

**Never modify existing migrations**. Create new numbered files in `backend/database/migrations/`.

**Embedded at compile**: Changes require rebuild. Use `//go:embed migrations/*.sql` in `database/db.go`.

**Schema changes**: Consider TimescaleDB constraints (hypertables cannot be altered like normal tables). Continuous aggregates require manual refresh policies.

### Frontend SSE Decoding

**Minified events** (`metrics_update`) must be decoded in `lib/sse.js`:
```javascript
const fieldMap = {
  s: 'server_id', t: 'timestamp', c: 'cpu_usage_percent',
  mt: 'memory_total_bytes', mu: 'memory_used_bytes', /* ... */
};
```

**When adding new metric fields**: Update both `backend/sse/broker.go` (minification) and `frontend/src/lib/sse.js` (decoding).

### Agent Runs as Unprivileged User

**Security principle**: Agent runs as system user `watchflare` (not root).

**Permissions**:
- Read: `/etc/watchflare/` (config owned by root)
- Write: `/var/lib/watchflare/` (owned by watchflare)

**Impact**:
- Cannot modify own binary or config
- Package collectors must use unprivileged commands (e.g., `brew list` works, `apt` requires sudo)
- Some system info may be unavailable (acceptable tradeoff)

### gRPC Timestamp Window

**Default: ±300 seconds (5 minutes)**. Configurable via `GRPC_TIMESTAMP_WINDOW` env var.

**Clock skew**: If agent and backend clocks differ by >5min, all RPCs fail. Monitor for "timestamp out of window" errors in logs.

**Time sync**: Agents should run NTP/timesyncd. Backend timestamp validation is defensive, not a replacement for proper time sync.

---

## Testing Notes

**Backend**: Tests use in-memory SQLite (`:memory:`). TimescaleDB features disabled in tests. Run with `go test ./...`.

**Agent WAL**: Tests create temp directories (`t.TempDir()`). Verify atomic truncation in `agent/wal/wal_test.go`.

**Package collectors**: Most tests are smoke tests (check name, availability). Full collection tests require specific CLI tools installed.

**Integration**: No automated integration tests yet. Manual testing uses local backend + agent in dev mode.

---

## Security Checklist for Code Changes

- [ ] Agent RPCs include `agent_id`, `agent_key`, `timestamp` fields
- [ ] Sensitive data (passwords, tokens, keys) never logged or returned in API responses
- [ ] File writes use restrictive permissions (0600 for keys, 0640 for configs)
- [ ] Database queries use parameterized statements (GORM handles this)
- [ ] JWT tokens validated on every protected endpoint (middleware already does this)
- [ ] TLS 1.3 enforced (check `MinVersion` and `MaxVersion` both set to `VersionTLS13`)
- [ ] Registration tokens hashed before storage (SHA-256, never plaintext in DB)
- [ ] HMAC comparisons use `hmac.Equal()` (constant-time, not `==`)

---

## Development Workflow

**Typical session**:
1. Start database: `docker compose up -d`
2. Start backend: `cd backend && go run .`
3. Start frontend: `cd frontend && npm run dev`
4. Agent (dev mode): `cd agent && ./watchflare-agent` (requires prior registration)

**Agent install/test cycle**:
1. Build: `go build -o watchflare-agent`
2. Stop service: `sudo launchctl bootout system/io.watchflare.agent`
3. Copy new binary: `sudo cp watchflare-agent /usr/local/bin/`
4. Start service: `sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist`
5. Check logs: `tail -f /var/log/watchflare-agent.log`

**Agent uninstall:**
```bash
# macOS
sudo ./uninstall-macos.sh   # Prompts for data/config/user removal

# Linux
sudo ./uninstall-linux.sh   # Prompts for data/config/user removal
```

**Database inspection**:
```bash
docker exec -it watchflare-postgres psql -U watchflare -d watchflare
\dt                                    # List tables
SELECT * FROM servers;                 # View servers
SELECT * FROM packages LIMIT 10;       # View packages
\d+ metrics                            # Describe metrics table (hypertable)
```

---

## Documentation

- `docs/architecture.md`: System overview, data flows, deployment
- `docs/internals.md`: Detailed component breakdown (every package/module)
- `docs/security.md`: Security model (TLS, HMAC, JWT, key management)
- `agent/INSTALL.md`: Agent installation guide (macOS)
- `agent/INSTALL-LINUX.md`: Agent installation guide (Linux/systemd)
- `README.md`: (Currently minimal)
