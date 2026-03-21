# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Watchflare — self-hosted server monitoring: **Backend** (Go, gRPC + HTTP), **Agent** (Go), **Frontend** (SvelteKit 5, SSE).
Data flow: Agents → gRPC/TLS 1.3 → Backend → PostgreSQL/TimescaleDB → SSE → Frontend

## Versioning

**Commit format — Conventional Commits:**
- `feat: description` — new feature
- `fix: description` — bug fix
- `chore:`, `docs:`, `refactor:`, `test:`, `ci:`, `style:` — maintenance

Scope is optional: `feat(agent): description`, `fix(backend): description`

**Release process:** see `docs/release-workflow.md`

## Build & Run

### Backend
```bash
cd backend
go run .                                # Dev
go build -o watchflare-backend          # Build (always use -o flag)
go test ./...                           # Tests (uses in-memory SQLite)
go test ./handlers -v                   # Single package
go test -run TestCreateAgent ./services # Single test
```
Env: copy `.env.example` → `.env`, set `JWT_SECRET` (>=32 chars). Test creds: `admin@watchflare.io` / `watchflare_p4ss`

### Agent
```bash
cd agent
go build -o watchflare-agent            # Build (always use -o flag)
go test ./...                           # Tests
./watchflare-agent register --token=wf_reg_... --host=localhost --port=50051
```

### Frontend
```bash
cd frontend
npm install && npm run dev              # Dev (http://localhost:5173)
npm run build                           # Production build
npm run test                            # Vitest
```

### Database
```bash
docker compose up -d                    # Start TimescaleDB
docker exec -it watchflare-postgres psql -U watchflare -d watchflare
```
Connection: `postgresql://watchflare:watchflare_dev@localhost:5433/watchflare`

### Dev session
1. `docker compose up -d` → 2. `cd backend && go run .` → 3. `cd frontend && npm run dev`

## Architecture (Key Decisions)

- **Heartbeats**: 5s agent → in-memory cache (no DB write) → SSE broadcast. DB sync every 5min. Stale after 15s.
- **SSE minification**: metric fields compressed to 1-2 chars in `backend/sse/broker.go`, decoded in `frontend/src/lib/sse.js` — both must be updated together
- **TimescaleDB continuous aggregates**: 10m/15m/2h/8h buckets for time ranges. Migrations embedded via `//go:embed`
- **Agent security**: runs as unprivileged `watchflare` user. HMAC-SHA256 per RPC, ±5min timestamp window
- **WAL**: append-only metrics buffer when backend unreachable, auto-replay on reconnect
- **Clock desync**: detected in gRPC interceptor, tracked in HeartbeatCache, shown as frontend banner

## Critical Patterns

- **Protobuf**: schema in `shared/proto/agent.proto`, generated Go code in `shared/proto/` (run `buf generate` or `protoc` to regenerate)
- **New RPC**: protobuf message must have `agent_id`, `agent_key`, `timestamp` fields for HMAC auth
- **New metric field**: update `backend/sse/broker.go` (minify) + `frontend/src/lib/sse.js` (decode)
- **New migration**: never modify existing files in `backend/database/migrations/`, create new numbered file
- **New package collector**: implement `Collector` interface in `agent/packages/`, register in `registry.go`
- **Frontend components**: Svelte 5 runes (`$props`, `$state`, `$derived`), bits-ui for dropdowns/selects

## Security Rules

- Tokens/keys: never log, never return in API responses
- File permissions: 0600 keys, 0640 configs
- HMAC: always `hmac.Equal()` (constant-time), never `==`
- TLS 1.3: `MinVersion` and `MaxVersion` both `VersionTLS13`
- Registration tokens: SHA-256 hashed before DB storage

## Key Entry Points

| Component | File | Purpose |
|-----------|------|---------|
| Backend bootstrap | `backend/main.go` | HTTP + gRPC + 3 background workers |
| Agent bootstrap | `agent/main.go` | register vs run mode |
| gRPC handlers | `backend/grpc/agent_service.go` | Register, Heartbeat, SendMetrics, SendPackageInventory |
| HTTP handlers | `backend/handlers/` | auth, servers, metrics, packages, sse |
| Metrics loop | `agent/wal/sender.go:Run()` | Collect → WAL → Send |
| Cache | `backend/cache/heartbeat.go` | In-memory heartbeat state |
| SSE broker | `backend/sse/broker.go` | Event broadcasting |

## Documentation

- `README.md` — project intro
- `SECURITY.md` — security policy
- `docs/` (local, gitignored) — architecture, internals, install guides, version history
- `.claude/rules/` — detailed supplementary rules (architecture, code style, testing, security, agent paths)
