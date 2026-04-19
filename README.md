# Watchflare

Self-hosted host monitoring with real-time dashboards. Lightweight agents report metrics over gRPC/TLS to a central backend, served as a single binary with an embedded web UI.

## Features

- **Real-time monitoring** — CPU, memory, disk, network, load average, temperature via SSE streaming
- **Docker container metrics** — Per-container CPU, memory, network tracking via Docker API
- **Lightweight agents** — Single binary, ~10MB, runs as a system service (Linux/macOS)
- **Secure by default** — TLS 1.3 (auto-generated PKI), HMAC-signed RPCs, JWT authentication
- **Package inventory** — Tracks installed packages across 28 package managers with daily delta detection
- **Write-ahead log** — Agents buffer metrics locally when the backend is unreachable
- **Single binary** — Frontend embedded in the Go backend via `go:embed`
- **TimescaleDB** — Automatic partitioning, compression, continuous aggregates, 30-day retention

## Architecture

```
Agents (Linux/macOS) (Windows support coming soon)
  │
  │ gRPC + TLS 1.3 + HMAC-SHA256
  ▼
Backend (Go)
  │
  ├── HTTP API + Embedded Frontend
  ├── SSE (real-time updates)
  └── TimescaleDB (metrics storage)
        │
        ▼
  Web Dashboard (SvelteKit)
```

## Quick Start

### Docker Compose (recommended)

```bash
git clone https://github.com/Kilian-Pichard/watchflare.git
cd watchflare

# Configure environment
cp .env.example .env
# Edit .env: set POSTGRES_PASSWORD and JWT_SECRET

# Start
docker compose -f docker-compose.yml up -d

# Open http://localhost:8080 and create your admin account
```

### Install an Agent

```bash
curl -sSL https://get.watchflare.io | sudo bash -s -- \
  --token=wf_reg_xxx --host=your-server --port=50051
```

### Development

```bash
# Start database
docker compose up -d

# Backend (terminal 1)
cd backend
cp .env.example .env
go run .

# Frontend (terminal 2)
cd frontend
npm install
npm run dev
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go, Gin, gRPC, GORM |
| Frontend | SvelteKit 5, Tailwind CSS, uPlot |
| Database | PostgreSQL + TimescaleDB |
| Agent | Go, gopsutil |
| Security | TLS 1.3, HMAC-SHA256, JWT, bcrypt |

## Documentation

Full documentation coming soon at [watchflare.io/docs](https://watchflare.io/docs).

## License

AGPL-3.0 — See [LICENSE](LICENSE) for details.
