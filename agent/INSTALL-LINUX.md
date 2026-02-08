# Watchflare Agent - Linux Installation Guide

## Prerequisites

- Linux distribution with systemd (Ubuntu 20.04+, Debian 11+, RHEL/CentOS 8+, Arch Linux)
- Administrator (sudo) access
- Go 1.21+ (for building from source)

## Installation Steps

### 1. Build the Agent

**Option A: Build all architectures (recommended)**
```bash
cd agent/
./build-all.sh
```

This creates binaries in `./dist/`:
```
dist/
├── watchflare_checksums.txt       # SHA-256 checksums
├── linux_amd64/watchflare-agent   # Linux x86-64
├── linux_arm64/watchflare-agent   # Linux ARM64
├── darwin_amd64/watchflare-agent  # macOS Intel
└── darwin_arm64/watchflare-agent  # macOS Apple Silicon
```

**Option B: Build for current platform only**
```bash
cd agent/
go build -o watchflare-agent
```

**Option C: Build for specific platform**
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o watchflare-agent-linux-amd64

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o watchflare-agent-linux-arm64
```

### 2. Run Installation Script

**Option A: Install + Register + Start in one command**

```bash
sudo ./install-linux.sh --token=YOUR_TOKEN --host=YOUR_HOST --port=50051
```

This will:
- Install the agent
- Register with your backend
- Start the service automatically

**Option B: Install only (register later)**

```bash
sudo ./install-linux.sh
```

This script will:
- Create a system user `watchflare` (if not exists)
- Create directories:
  - `/etc/watchflare/` (configuration)
  - `/var/lib/watchflare/` (data, WAL, package state)
- Install binary to `/usr/local/bin/watchflare-agent`
- Install systemd service to `/etc/systemd/system/watchflare-agent.service`
- Set proper permissions (principle of least privilege)

### 3. Register the Agent (if not done during installation)

If you chose Option B above, register the agent with your backend:

```bash
sudo /usr/local/bin/watchflare-agent register \
  --token=YOUR_REGISTRATION_TOKEN \
  --host=YOUR_BACKEND_HOST \
  --port=50051
```

The registration command will:
- Connect to your Watchflare backend
- Exchange the registration token for agent credentials
- Save configuration to `/etc/watchflare/agent.conf`
- Download TLS CA certificate to `/etc/watchflare/ca.pem`

### 4. Start the Service (if not done during installation)

If you registered manually, start the service:

```bash
sudo systemctl enable watchflare-agent
sudo systemctl start watchflare-agent
```

## Service Management

**Check Status:**
```bash
sudo systemctl status watchflare-agent
```

**View Logs:**
```bash
# Systemd journal
journalctl -u watchflare-agent -f

# Or direct log file
tail -f /var/log/watchflare-agent.log
```

**Stop Service:**
```bash
sudo systemctl stop watchflare-agent
```

**Start Service:**
```bash
sudo systemctl start watchflare-agent
```

**Restart Service:**
```bash
sudo systemctl restart watchflare-agent
```

**Enable at boot:**
```bash
sudo systemctl enable watchflare-agent
```

**Disable at boot:**
```bash
sudo systemctl disable watchflare-agent
```

## File Locations

| Path | Purpose | Owner | Permissions |
|------|---------|-------|-------------|
| `/usr/local/bin/watchflare-agent` | Binary | root:root | 755 |
| `/etc/watchflare/` | Configuration | root:watchflare | 750 |
| `/etc/watchflare/agent.conf` | Config file (contains credentials) | root:watchflare | 640 |
| `/etc/watchflare/ca.pem` | TLS CA certificate | root:watchflare | 644 |
| `/var/lib/watchflare/` | Data directory | watchflare:watchflare | 750 |
| `/var/lib/watchflare/wal/` | Write-Ahead Log | watchflare:watchflare | 750 |
| `/var/lib/watchflare/packages.state.json` | Package inventory state | watchflare:watchflare | 640 |
| `/etc/systemd/system/watchflare-agent.service` | Service definition | root:root | 644 |
| `/var/log/watchflare-agent.log` | Agent logs | watchflare:watchflare | 644 |

## Security

The agent runs as a dedicated system user `watchflare` with minimal privileges:
- ✅ **No root access** - Runs as unprivileged user
- ✅ **No shell** - User has `/usr/sbin/nologin` as shell
- ✅ **No home directory** - User home is `/var/empty`
- ✅ **Restricted permissions** - Can only access its own files
- ✅ **TLS encryption** - All communication with backend is encrypted
- ✅ **HMAC authentication** - Requests are authenticated with agent key
- ✅ **Systemd hardening** - NoNewPrivileges, PrivateTmp, ProtectSystem enabled

## Uninstallation

To completely remove the agent:

```bash
sudo ./uninstall-linux.sh
```

This script will:
1. Stop and disable the service
2. Remove the binary
3. Prompt to remove data directory (WAL, package state)
4. Prompt to remove configuration (agent credentials)
5. Prompt to remove system user `watchflare`
6. Prompt to remove log files

**Note:** The script asks for confirmation before removing data and configuration to prevent accidental data loss.

## Troubleshooting

### Agent won't start

Check logs:
```bash
journalctl -u watchflare-agent -n 50
# Or
tail -50 /var/log/watchflare-agent.log
```

Common issues:
- **"config file not found"** → Run registration command first
- **"permission denied"** → Check file ownership and permissions
- **"connection refused"** → Verify backend host and port
- **"invalid agent credentials"** → Re-register the agent

### Permission errors

Verify ownership:
```bash
ls -la /etc/watchflare/
ls -la /var/lib/watchflare/
```

Fix permissions:
```bash
sudo chown -R root:watchflare /etc/watchflare/
sudo chown -R watchflare:watchflare /var/lib/watchflare/
```

### Check if service is loaded

```bash
systemctl list-units | grep watchflare
```

Should output:
```
watchflare-agent.service  loaded active running Watchflare Monitoring Agent
```

### Verify agent user exists

```bash
id watchflare
```

Should output user and group info.

## Updating the Agent

1. Stop the service:
   ```bash
   sudo systemctl stop watchflare-agent
   ```

2. Build new version:
   ```bash
   cd agent/
   go build -o watchflare-agent
   ```

3. Copy new binary:
   ```bash
   sudo cp watchflare-agent /usr/local/bin/watchflare-agent
   sudo chown root:root /usr/local/bin/watchflare-agent
   sudo chmod 755 /usr/local/bin/watchflare-agent
   ```

4. Start the service:
   ```bash
   sudo systemctl start watchflare-agent
   ```

## Configuration Reference

The configuration file `/etc/watchflare/agent.conf` is in TOML format:

```toml
# Backend connection
server_host = "backend.example.com"
server_port = "50051"

# Agent credentials (generated during registration)
agent_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
agent_key = "base64encodedkey..."

# TLS configuration
ca_cert_file = "/etc/watchflare/ca.pem"
server_name = "watchflare"

# Intervals (seconds)
heartbeat_interval = 5   # Heartbeat every 5 seconds
metrics_interval = 30    # Collect metrics every 30 seconds

# Write-Ahead Log (WAL) - Enabled by default (optional, set to false to disable)
wal_enabled = true
wal_path = "/var/lib/watchflare/wal/metrics.wal"
wal_max_size_mb = 10
```

## Package Collection

The agent automatically collects package inventory:
- **Initial scan**: 60 seconds after agent startup
- **Daily scan**: Every day at 3:00 AM
- **Delta updates**: Only changes (added/removed/updated) are sent
- **State file**: `/var/lib/watchflare/packages.state.json`

Supported package managers on Linux:
- ✅ **APT** (`apt`, `dpkg`) - Debian, Ubuntu
- ✅ **YUM/DNF** (`yum`, `dnf`, `rpm`) - RHEL, CentOS, Fedora, AlmaLinux
- ✅ **Pacman** (`pacman`) - Arch Linux
- ✅ **Snap** (`snap`) - Ubuntu and others
- ✅ **Flatpak** (`flatpak`) - Cross-distribution
- ✅ **pip** (Python packages)
- ✅ **npm** (Node.js packages)
- ✅ **gem** (Ruby packages)
- ✅ **cargo** (Rust packages)
- ✅ **go** (Go modules)

## Distribution-Specific Notes

### Ubuntu/Debian

Default shell for system users is `/usr/sbin/nologin`.

### RHEL/CentOS/AlmaLinux

If SELinux is enabled, you may need to set proper contexts:
```bash
sudo restorecon -Rv /usr/local/bin/watchflare-agent
sudo restorecon -Rv /etc/watchflare
sudo restorecon -Rv /var/lib/watchflare
```

### Arch Linux

Install dependencies:
```bash
sudo pacman -S go
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/yourusername/watchflare
- Documentation: https://watchflare.example.com/docs
