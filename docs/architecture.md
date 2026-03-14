# Watchflare — Architecture

Watchflare is a self-hosted server monitoring platform. A lightweight agent runs on
each monitored server, collects system metrics and package inventory, and transmits
them to a central backend over gRPC. A web dashboard displays real-time status,
historical metric charts, and package details, driven by Server-Sent Events.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Monitored Servers                         │
│                                                             │
│  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐           │
│  │ Agent  │  │ Agent  │  │ Agent  │  │ Agent  │           │
│  └───┬────┘  └───┬────┘  └───┬────┘  └───┬────┘           │
└──────┼───────────┼───────────┼───────────┼─────────────────┘
       │           │           │           │
       │  gRPC / TLS 1.3       │           │
       ▼           ▼           ▼           ▼
┌─────────────────────────────────────────────────────────────┐
│                      Backend (Go)                           │
│                                                             │
│  ┌──────────┐   ┌──────────┐   ┌─────────┐   ┌─────────┐  │
│  │  gRPC    │   │  HTTP    │   │  Cache  │   │  SSE    │  │
│  │  Server  │   │  (Gin)   │   │  Layer  │   │  Broker │  │
│  └────┬─────┘   └────┬─────┘   └────┬────┘   └────┬────┘  │
│       │              │              │              │        │
│       ▼              ▼              ▼              │        │
│  ┌──────────────────────────────────────┐         │        │
│  │    PostgreSQL / TimescaleDB          │         │        │
│  │  (metrics, servers, packages, users) │         │        │
│  └──────────────────────────────────────┘         │        │
└────────────────────────────────────────────────────┼────────┘
                                                     │ SSE
                                                     ▼
                                          ┌──────────────────┐
                                          │  Frontend        │
                                          │  (SvelteKit)     │
                                          │  Dashboard / UI  │
                                          └──────────────────┘
```

Three binaries, one database, one browser connection:

| Component | Language | Ports | Role |
|-----------|----------|-------|------|
| Backend | Go | 8080 (HTTP), 50051 (gRPC) | Central server, API, gRPC endpoint, SSE hub |
| Agent | Go | — (outbound only) | Runs on each monitored server |
| Frontend | Svelte | 5173 (dev) | Dashboard served to the operator's browser |
| Database | PostgreSQL + TimescaleDB | 5432 | Persistent store, time-series hypertable |

---

## Data Flows

### 1. Agent Registration

Registration bootstraps trust. The agent starts with no shared secret — the
registration token is the root of trust.

```
Admin (browser)          Backend                      Agent
      │                    │                            │
      │ POST /servers      │                            │
      │ {name, ip}         │                            │
      │───────────────────>│                            │
      │                    │  generates:                │
      │                    │    token = wf_reg_<32hex> │
      │                    │    key   = 32 random bytes│
      │                    │    stores SHA256(token)   │
      │ ← token, key      │                            │
      │                    │                            │
      │  (operator pastes  │                            │
      │   token into the   │                            │
      │   install command) │                            │
      │                    │                            │
      │                    │  RegisterServer RPC        │
      │                    │  { token, hostname,       │
      │                    │    IPs, platform, … }     │
      │                    │<───────────────────────────│
      │                    │                            │
      │                    │  hash(token) → DB lookup  │
      │                    │  check expiry (24 h)      │
      │                    │  check status == pending  │
      │                    │  validate IP if set       │
      │                    │  clear token, status→online
      │                    │                            │
      │                    │  ← { agent_id,           │
      │                    │      agent_key,           │
      │                    │      ca_cert (PEM) }      │
      │                    │───────────────────────────>│
      │                    │                            │
      │                    │   agent saves credentials │
      │                    │   + CA cert to disk       │
```

Key points:
- The plaintext token is shown once and never persisted. Only `SHA256(token)` lives in the DB.
- The agent connects with `InsecureSkipVerify` for this single RPC (no CA cert yet).
- After registration the agent pins the received CA cert — all future TLS is strict.
- Token expires after 24 hours if unused.

---

### 2. Heartbeat & Online / Offline Detection

```
Agent                   Backend                      Frontend
  │                       │                             │
  │  every 5 s            │                             │
  │  Heartbeat RPC        │                             │
  │  { agent_id, IPs }    │                             │
  │  + HMAC auth          │                             │
  │──────────────────────>│                             │
  │                       │  HeartbeatCache.Update()    │
  │                       │  (in-memory only)           │
  │                       │                             │
  │                       │  SSE → server_update        │
  │                       │────────────────────────────>│
  │                       │                             │
  │  ····· agent stops ······                           │
  │                       │                             │
  │                       │  StaleChecker (every 10 s) │
  │                       │  no heartbeat > 15 s        │
  │                       │  → status = "offline"       │
  │                       │  SSE → server_update        │
  │                       │────────────────────────────>│
  │                       │                             │
  │                       │  SyncWorker (every 5 min)   │
  │                       │  flush cache → DB           │
  │                       │  (last_seen, status, IPs)   │
```

The cache layer decouples three different rates:
- **5 s** — heartbeat from agent (network + CPU cheap, no DB write)
- **10 s** — stale check interval (in-memory scan)
- **5 min** — database flush (batched writes)

---

### 3. Metrics Collection & Streaming

```
Agent                   Backend                      Frontend
  │                       │                             │
  │  every 30 s           │                             │
  │  1. Collect metrics   │                             │
  │  2. Append → WAL      │                             │
  │  3. SendMetrics RPC   │                             │
  │     + HMAC auth       │                             │
  │──────────────────────>│                             │
  │                       │  INSERT INTO metrics        │
  │                       │  (TimescaleDB hypertable)   │
  │                       │                             │
  │                       │  SSE → metrics_update       │
  │                       │  (minified: single-letter   │
  │                       │   field keys)               │
  │                       │────────────────────────────>│
  │  4. Clear WAL         │                             │
  │     (only on success) │                             │
  │                       │                             │
  │                       │  AggregatedMetrics          │
  │                       │  Scheduler (every 30 s)     │
  │                       │  AVG(cpu), SUM(mem, disk)   │
  │                       │  across all online servers  │
  │                       │  SSE → aggregated_update    │
  │                       │────────────────────────────>│
```

If the backend is unreachable, metrics accumulate in the agent's WAL. On the next
successful connection, all pending records are replayed in order before new metrics
are sent. The WAL is only cleared after every pending record succeeds.

---

### 4. Package Inventory

```
Agent                         Backend
  │                              │
  │  daily at 03:00              │
  │  1. CollectAll()             │
  │     (30+ package managers)   │
  │  2. Load previous state      │
  │  3. ComputeDelta()           │
  │     → added / removed /      │
  │       updated                │
  │                              │
  │  First run → full inventory  │
  │  SendPackageInventory        │
  │  { all_packages[] }          │
  │─────────────────────────────>│  upsert packages
  │  Save new state to disk      │  create history records
  │                              │  record collection meta
  │                              │
  │  Subsequent → delta only     │
  │  SendPackageInventory        │
  │  { added[], removed[],       │
  │    updated[] }               │
  │─────────────────────────────>│  process changes
  │  Save new state to disk      │  in single transaction
```

First run sends everything so the backend has a full baseline. Every run after that
sends only the diff, keeping payloads small.

---

### 5. Web User Authentication

```
Browser                   Backend
  │                         │
  │  POST /auth/login       │
  │  { email, password }    │
  │────────────────────────>│
  │                         │  bcrypt.Compare()
  │                         │  generate JWT (exp: 7 d)
  │  ← Set-Cookie:          │
  │     jwt_token           │
  │     HttpOnly, SameSite  │
  │<────────────────────────│
  │                         │
  │  (every protected req)  │
  │  Cookie: jwt_token      │
  │────────────────────────>│
  │                         │  verify signature
  │                         │  check expiry
  │                         │  confirm user in DB
  │                         │  set user_id in context
```

Only one user can register (the first). `GET /auth/setup-required` tells the UI
whether to show the registration page or the login page.

---

## Design Principles

**1. Durability first.** Metrics are written to a WAL before any network call.
If the backend is down, nothing is lost — records replay on reconnection.

**2. Minimize database pressure.** Heartbeats update an in-memory cache at 5 s.
The database is written every 5 min (SyncWorker). Stale detection runs on the
cache, not the DB.

**3. Environment-aware collection.** Containers skip disk metrics (shared with the
host, would double-count). VMs skip temperature (no physical sensor access). Each
environment type maps to a tailored `MetricsConfig`.

**4. Delta over full.** Package inventory sends a full baseline once, then only
changes. Keeps daily payloads small regardless of package count.

**5. Real-time without polling.** SSE pushes all live updates to the browser.
The dashboard never polls for status or current metrics.

**6. Two-level aggregation.** Dashboard charts first average metrics per server
per time bucket, then aggregate across servers (SUM for memory/disk, AVG for CPU).
Prevents inflation when a server reports multiple times in one interval.

**7. Crash-safe file operations.** WAL truncation writes to a temp file, syncs it,
syncs the directory, then does an atomic rename. On a crash at any point, exactly
one valid file survives.

**8. One-time secrets.** Registration tokens are hashed (SHA-256) before storage.
The plaintext is shown once and never persisted. Agent keys are generated randomly
and returned once during registration.

---

## Deployment

`docker-compose.yml` runs the database (TimescaleDB on PostgreSQL 16).
Backend and frontend run as separate processes (or containers).

```
# Start database
docker compose up -d

# Start backend  (reads .env for configuration)
cd backend && go run .

# Start frontend (dev server)
cd frontend && npm run dev
```

The backend auto-generates TLS certificates on first startup (`./pki/`).
No manual certificate setup is required unless you want to supply your own.
