# Watchflare Agent - Installation Guide

## macOS Installation

### Prerequisites

- macOS 10.13 (High Sierra) or later
- Administrator (sudo) access
- Go 1.21+ (for building from source)

### Installation Steps

#### 1. Build the Agent

```bash
cd agent/
go build -o watchflare-agent
```

#### 2. Run Installation Script

**Option A: Install + Register + Start in one command**

```bash
sudo ./install-macos.sh --token=YOUR_TOKEN --host=YOUR_HOST --port=50051
```

This will:
- Install the agent
- Register with your backend
- Start the service automatically

**Option B: Install only (register later)**

```bash
sudo ./install-macos.sh
```

This script will:
- Create a system user `watchflare` (if not exists)
- Create directories:
  - `/etc/watchflare/` (configuration)
  - `/var/lib/watchflare/` (data, WAL, package state)
- Install binary to `/usr/local/bin/watchflare-agent`
- Create LaunchDaemon at `/Library/LaunchDaemons/io.watchflare.agent.plist`
- Set proper permissions (principle of least privilege)

#### 3. Register the Agent (if not done during installation)

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

#### 4. Start the Service (if not done during installation)

If you registered manually, start the service:

```bash
sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist
```

### Service Management

**Check Status:**
```bash
sudo launchctl print system/io.watchflare.agent
```

**View Logs:**
```bash
tail -f /var/log/watchflare-agent.log
```

**Stop Service:**
```bash
sudo launchctl bootout system/io.watchflare.agent
```

**Start Service:**
```bash
sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist
```

**Restart Service:**
```bash
sudo launchctl bootout system/io.watchflare.agent
sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist
```

### File Locations

| Path | Purpose | Owner | Permissions |
|------|---------|-------|-------------|
| `/usr/local/bin/watchflare-agent` | Binary | root:wheel | 755 |
| `/etc/watchflare/` | Configuration | root:staff | 750 |
| `/etc/watchflare/agent.conf` | Config file (contains credentials) | root:staff | 640 |
| `/etc/watchflare/ca.pem` | TLS CA certificate | root:staff | 644 |
| `/var/lib/watchflare/` | Data directory | watchflare:staff | 750 |
| `/var/lib/watchflare/wal/` | Write-Ahead Log | watchflare:staff | 750 |
| `/var/lib/watchflare/packages.state.json` | Package inventory state | watchflare:staff | 640 |
| `/Library/LaunchDaemons/io.watchflare.agent.plist` | Service definition | root:wheel | 644 |
| `/var/log/watchflare-agent.log` | Agent logs | watchflare:staff | 644 |

### Security

The agent runs as a dedicated system user `watchflare` with minimal privileges:
- ✅ **No root access** - Runs as unprivileged user
- ✅ **No shell** - User has `/usr/bin/false` as shell
- ✅ **No home directory** - User home is `/var/empty`
- ✅ **Restricted permissions** - Can only access its own files
- ✅ **TLS encryption** - All communication with backend is encrypted
- ✅ **HMAC authentication** - Requests are authenticated with agent key

### Uninstallation

To completely remove the agent:

```bash
sudo ./uninstall-macos.sh
```

This script will:
1. Stop and remove the service
2. Remove the binary
3. Prompt to remove data directory (WAL, package state)
4. Prompt to remove configuration (agent credentials)
5. Prompt to remove system user `watchflare`
6. Prompt to remove log files

**Note:** The script asks for confirmation before removing data and configuration to prevent accidental data loss.

### Troubleshooting

#### Agent won't start

Check logs:
```bash
tail -50 /var/log/watchflare-agent.log
```

Common issues:
- **"config file not found"** → Run registration command first
- **"permission denied"** → Check file ownership and permissions
- **"connection refused"** → Verify backend host and port
- **"invalid agent credentials"** → Re-register the agent

#### Permission errors

Verify ownership:
```bash
ls -la /etc/watchflare/
ls -la /var/lib/watchflare/
```

Fix permissions:
```bash
sudo chown -R root:staff /etc/watchflare/
sudo chown -R watchflare:staff /var/lib/watchflare/
```

#### Check if service is loaded

```bash
sudo launchctl list | grep watchflare
```

Should output:
```
-	0	io.watchflare.agent
```

#### Verify agent user exists

```bash
dscl . -read /Users/watchflare
```

### Updating the Agent

1. Stop the service:
   ```bash
   sudo launchctl bootout system/io.watchflare.agent
   ```

2. Build new version:
   ```bash
   cd agent/
   go build -o watchflare-agent
   ```

3. Copy new binary:
   ```bash
   sudo cp watchflare-agent /usr/local/bin/watchflare-agent
   sudo chown root:wheel /usr/local/bin/watchflare-agent
   sudo chmod 755 /usr/local/bin/watchflare-agent
   ```

4. Start the service:
   ```bash
   sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist
   ```

### Configuration Reference

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
wal_max_size_mb = 20
```

### Package Collection

The agent automatically collects package inventory:
- **Initial scan**: 60 seconds after agent startup
- **Daily scan**: Every day at 3:00 AM
- **Delta updates**: Only changes (added/removed/updated) are sent
- **State file**: `/var/lib/watchflare/packages.state.json`

Supported package managers on macOS:
- ✅ **Homebrew** (`brew`)

### Support

For issues or questions:
- GitHub Issues: https://github.com/yourusername/watchflare
- Documentation: https://watchflare.example.com/docs
