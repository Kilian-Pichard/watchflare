# Watchflare Agent - Installation Guide

## 📋 Prerequisites

- Linux system (Ubuntu, Debian, CentOS, RHEL, etc.)
- Root access or sudo privileges
- systemd (for service management)

## 🚀 Quick Installation

### 1. Build the Agent

```bash
cd agent
make build
```

### 2. Install

```bash
sudo ./install.sh
```

### 3. Configure and Start

```bash
# Register the agent with your backend
sudo watchflare-agent --token YOUR_REGISTRATION_TOKEN --host your-backend.com --port 50051

# Start the service
sudo systemctl start watchflare-agent

# Check status
sudo systemctl status watchflare-agent
```

## 📁 Directory Structure

The installation follows the [Filesystem Hierarchy Standard (FHS)](https://refspecs.linuxfoundation.org/FHS_3.0/fhs/index.html):

```
/etc/watchflare/              # Configuration (root:watchflare 750)
├── agent.conf                # Main configuration file (640)

/var/lib/watchflare/          # Variable data (watchflare:watchflare 750)
├── logs/                     # Application logs
├── run/                      # Runtime files (PID, sockets)
└── cache/                    # Cache data (700)

/var/log/watchflare/          # System logs (watchflare:watchflare 750)
└── watchflare-agent.log      # Main log file

/usr/bin/                     # Binaries (root:root 755)
└── watchflare-agent          # Main executable
```

## 🔒 Security

### System User

The agent runs as a dedicated system user `watchflare` with minimal privileges:
- No home directory
- No login shell (`/bin/false`)
- Limited filesystem access

### File Permissions

| Path | Owner | Permissions | Description |
|------|-------|-------------|-------------|
| `/etc/watchflare` | root:watchflare | 750 (rwxr-x---) | Config dir |
| `/etc/watchflare/agent.conf` | root:watchflare | 640 (rw-r-----) | Config file |
| `/var/lib/watchflare` | watchflare:watchflare | 750 (rwxr-x---) | Data dir |
| `/var/lib/watchflare/cache` | watchflare:watchflare | 700 (rwx------) | Cache dir |
| `/usr/bin/watchflare-agent` | root:root | 755 (rwxr-xr-x) | Binary |

### systemd Hardening

The service includes security features:
- `NoNewPrivileges=true` - Prevents privilege escalation
- `PrivateTmp=true` - Isolated /tmp directory
- `ProtectSystem=strict` - Read-only access to most of the filesystem
- `ProtectHome=true` - No access to user home directories
- `ReadWritePaths` - Explicitly defines writable paths

## 🔧 Configuration

### Environment Variables

You can override default paths using environment variables:

```bash
# Custom configuration directory
export WATCHFLARE_CONFIG_DIR=/custom/path/config

# Custom data directory
export WATCHFLARE_DATA_DIR=/custom/path/data

# Custom log directory
export WATCHFLARE_LOG_DIR=/custom/path/logs
```

### Service Management

```bash
# Start the agent
sudo systemctl start watchflare-agent

# Stop the agent
sudo systemctl stop watchflare-agent

# Restart the agent
sudo systemctl restart watchflare-agent

# Enable on boot
sudo systemctl enable watchflare-agent

# Disable on boot
sudo systemctl disable watchflare-agent

# View status
sudo systemctl status watchflare-agent

# View logs
sudo journalctl -u watchflare-agent -f
```

### Log Rotation

Logs are automatically rotated daily using logrotate:
- Keep 7 days of logs
- Compress old logs
- Create new log files with correct permissions

Configuration: `/etc/logrotate.d/watchflare`

## 🗑️ Uninstallation

```bash
sudo ./uninstall.sh
```

This will:
1. Stop and disable the service
2. Remove the systemd service file
3. Remove the binary
4. Remove logrotate configuration
5. Optionally remove data directories
6. Remove the system user

## 🔍 Troubleshooting

### Check Service Status

```bash
sudo systemctl status watchflare-agent
```

### View Logs

```bash
# Live logs
sudo journalctl -u watchflare-agent -f

# Last 100 lines
sudo journalctl -u watchflare-agent -n 100

# Application log file
sudo tail -f /var/log/watchflare/watchflare-agent.log
```

### Permissions Issues

If you encounter permission errors:

```bash
# Reset permissions
sudo chown -R root:watchflare /etc/watchflare
sudo chmod 750 /etc/watchflare
sudo chmod 640 /etc/watchflare/agent.conf

sudo chown -R watchflare:watchflare /var/lib/watchflare
sudo chmod -R 750 /var/lib/watchflare

sudo chown -R watchflare:watchflare /var/log/watchflare
sudo chmod 750 /var/log/watchflare
```

### Agent Not Starting

1. Check if directories exist:
```bash
ls -la /etc/watchflare
ls -la /var/lib/watchflare
ls -la /var/log/watchflare
```

2. Check configuration file:
```bash
sudo cat /etc/watchflare/agent.conf
```

3. Check binary permissions:
```bash
ls -la /usr/bin/watchflare-agent
```

### Manual Registration

If you need to re-register:

```bash
# Stop the service
sudo systemctl stop watchflare-agent

# Remove old configuration
sudo rm -f /etc/watchflare/agent.conf

# Register again
sudo watchflare-agent --token NEW_TOKEN --host backend.example.com

# Start the service
sudo systemctl start watchflare-agent
```

## 🎯 Best Practices

1. **Always use systemd** to manage the agent (don't run it manually)
2. **Monitor logs** regularly for any issues
3. **Keep the agent updated** to get security fixes
4. **Backup configuration** before making changes
5. **Use strong tokens** for registration
6. **Restrict network access** to the backend server only
7. **Regular security audits** of file permissions

## 📚 Additional Resources

- [systemd Documentation](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [FHS Standard](https://refspecs.linuxfoundation.org/FHS_3.0/fhs/index.html)
- [Linux Security Best Practices](https://www.cisecurity.org/cis-benchmarks/)
