# Installation Guide

## Prerequisites

- Docker and Docker Compose v2+
- A server with ports 8080 (HTTP) and 50051 (gRPC) available

## 1. Deploy the Backend

```bash
git clone <your-repo-url>
cd watchflare
```

Create your environment file:

```bash
cp .env.example .env
```

Edit `.env` and set secure values:

```env
POSTGRES_PASSWORD=your_secure_database_password
JWT_SECRET=your_random_jwt_secret
```

Generate a random JWT secret (use hex to avoid special characters):

```bash
openssl rand -hex 32
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_PASSWORD` | *required* | Database password |
| `JWT_SECRET` | *required* | JWT signing key (min 32 chars) |
| `POSTGRES_USER` | `watchflare` | Database user |
| `POSTGRES_DB` | `watchflare` | Database name |
| `PORT` | `8080` | HTTP port (frontend + API) |
| `GRPC_PORT` | `50051` | gRPC port (agents) |
| `TLS_MODE` | `auto` | `auto` (self-signed) or `custom` (bring your own certs) |
| `COOKIE_DOMAIN` | *(empty)* | Set to your domain when using HTTPS (enables secure cookies) |
| `GRPC_TIMESTAMP_WINDOW` | `300` | HMAC timestamp tolerance in seconds |

**Cookie security:** By default, cookies are not marked as `Secure` — this allows HTTP access (e.g., via IP address). When you set up a reverse proxy with HTTPS, set `COOKIE_DOMAIN=your-domain.com` in `.env` to enable secure cookies.

Start the stack:

```bash
docker compose -f docker-compose.prod.yml up -d
```

Verify both containers are running:

```bash
docker compose -f docker-compose.prod.yml ps
```

Open `http://your-server:8080` in your browser and create your admin account.

## 2. Add a Server

1. In the dashboard, click **Add Server**
2. Enter a name and optionally a fixed IP
3. Copy the registration token

## 3. Install an Agent

On the machine you want to monitor, download and install the agent.

### Linux (systemd)

```bash
# Download the agent binary
curl -L https://github.com/Kilian-Pichard/watchflare/releases/latest/download/watchflare-agent-linux-amd64 \
  -o watchflare-agent
chmod +x watchflare-agent

# Install as a system service (register + start in one command)
sudo ./install-linux.sh \
  --token=wf_reg_YOUR_TOKEN \
  --host=YOUR_BACKEND_IP \
  --port=50051
```

### macOS (launchd)

```bash
# Download the agent binary
curl -L https://github.com/Kilian-Pichard/watchflare/releases/latest/download/watchflare-agent-darwin-arm64 \
  -o watchflare-agent
chmod +x watchflare-agent

# Install as a system service
sudo ./install-macos.sh \
  --token=wf_reg_YOUR_TOKEN \
  --host=YOUR_BACKEND_IP \
  --port=50051
```

The agent will appear as "online" in the dashboard within a few seconds.

### Agent Management

**Linux:**
```bash
sudo systemctl status watchflare-agent    # Status
sudo systemctl restart watchflare-agent   # Restart
journalctl -u watchflare-agent -f         # Logs
```

**macOS:**
```bash
sudo launchctl bootout system/io.watchflare.agent    # Stop
sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist  # Start
tail -f /var/log/watchflare-agent.log                # Logs
```

## 4. Reverse Proxy (HTTPS)

For production, put the HTTP port (8080) behind a reverse proxy with TLS. The gRPC port (50051) uses its own TLS and should be exposed directly.

After setting up HTTPS, add `COOKIE_DOMAIN=your-domain.com` to `.env` and restart to enable secure cookies.

### Caddy

```
watchflare.example.com {
    reverse_proxy localhost:8080
}
```

### Nginx

```nginx
server {
    listen 443 ssl;
    server_name watchflare.example.com;

    ssl_certificate     /etc/letsencrypt/live/watchflare.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/watchflare.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE support
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
    }
}
```

## 5. Updating

```bash
cd watchflare
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

Data is persisted in Docker volumes (`pgdata` for the database, `pki_data` for TLS certificates).

## 6. Uninstalling

### Backend

```bash
docker compose -f docker-compose.prod.yml down

# To also remove data:
docker compose -f docker-compose.prod.yml down -v
```

### Agent

```bash
# Linux
sudo ./uninstall-linux.sh

# macOS
sudo ./uninstall-macos.sh
```
