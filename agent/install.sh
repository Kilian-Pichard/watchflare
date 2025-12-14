#!/bin/bash
# Watchflare Agent Installation Script
# Professional installation following Linux FHS standards

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WATCHFLARE_USER="watchflare"
WATCHFLARE_GROUP="watchflare"
INSTALL_DIR="/usr/bin"
CONFIG_DIR="/etc/watchflare"
DATA_DIR="/var/lib/watchflare"
LOG_DIR="/var/log/watchflare"
SERVICE_FILE="/etc/systemd/system/watchflare-agent.service"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   Watchflare Agent - Professional Installation${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}✗ This script must be run as root${NC}"
    echo -e "${YELLOW}  Please run: sudo $0${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Running as root"

# Step 1: Create system user
echo
echo -e "${BLUE}[1/7]${NC} Creating system user..."
if id "$WATCHFLARE_USER" &>/dev/null; then
    echo -e "${YELLOW}ℹ${NC}  User '$WATCHFLARE_USER' already exists"
else
    useradd --system --no-create-home --shell /bin/false "$WATCHFLARE_USER"
    echo -e "${GREEN}✓${NC} User '$WATCHFLARE_USER' created"
fi

# Step 2: Create directory structure
echo
echo -e "${BLUE}[2/7]${NC} Creating directory structure..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"/{logs,run,cache}
mkdir -p "$LOG_DIR"
echo -e "${GREEN}✓${NC} Directories created"

# Step 3: Set permissions
echo
echo -e "${BLUE}[3/7]${NC} Configuring permissions..."

# Config directory: root owns, watchflare group can read
chown -R root:$WATCHFLARE_GROUP "$CONFIG_DIR"
chmod 750 "$CONFIG_DIR"

# Data directory: watchflare owns everything
chown -R $WATCHFLARE_USER:$WATCHFLARE_GROUP "$DATA_DIR"
chmod -R 750 "$DATA_DIR"
chmod 700 "$DATA_DIR/cache"  # Cache is more restrictive

# Log directory: watchflare owns
chown -R $WATCHFLARE_USER:$WATCHFLARE_GROUP "$LOG_DIR"
chmod 750 "$LOG_DIR"

echo -e "${GREEN}✓${NC} Permissions configured"

# Step 4: Install binary
echo
echo -e "${BLUE}[4/7]${NC} Installing binary..."
if [ -f "./watchflare-agent" ]; then
    cp ./watchflare-agent "$INSTALL_DIR/watchflare-agent"
    chmod 755 "$INSTALL_DIR/watchflare-agent"
    chown root:root "$INSTALL_DIR/watchflare-agent"
    echo -e "${GREEN}✓${NC} Binary installed to $INSTALL_DIR"
else
    echo -e "${RED}✗ Binary not found${NC}"
    echo -e "${YELLOW}  Please build the agent first: make build${NC}"
    exit 1
fi

# Step 5: Create systemd service
echo
echo -e "${BLUE}[5/7]${NC} Installing systemd service..."
cat > "$SERVICE_FILE" <<'EOL'
[Unit]
Description=Watchflare Agent - Server Monitoring
Documentation=https://github.com/watchflare/watchflare
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=watchflare
Group=watchflare
ExecStart=/usr/bin/watchflare-agent
Restart=always
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/watchflare /var/log/watchflare /etc/watchflare

# Logging
StandardOutput=append:/var/log/watchflare/watchflare-agent.log
StandardError=append:/var/log/watchflare/watchflare-agent.log

[Install]
WantedBy=multi-user.target
EOL

echo -e "${GREEN}✓${NC} Systemd service created"

# Step 6: Install logrotate configuration
echo
echo -e "${BLUE}[6/7]${NC} Installing logrotate configuration..."
cat > /etc/logrotate.d/watchflare <<'EOL'
/var/log/watchflare/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 640 watchflare watchflare
    sharedscripts
    postrotate
        systemctl reload watchflare-agent >/dev/null 2>&1 || true
    endscript
}
EOL

echo -e "${GREEN}✓${NC} Logrotate configuration installed"

# Step 7: Reload systemd and enable service
echo
echo -e "${BLUE}[7/7]${NC} Configuring service..."
systemctl daemon-reload
systemctl enable watchflare-agent
echo -e "${GREEN}✓${NC} Service enabled (will start on boot)"

# Final summary
echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ Installation completed successfully!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Configure the agent with your registration token:"
echo -e "     ${BLUE}watchflare-agent --token YOUR_TOKEN --host YOUR_BACKEND_HOST${NC}"
echo
echo -e "  2. Start the service:"
echo -e "     ${BLUE}sudo systemctl start watchflare-agent${NC}"
echo
echo -e "  3. Check status:"
echo -e "     ${BLUE}sudo systemctl status watchflare-agent${NC}"
echo
echo -e "${YELLOW}Directory Structure:${NC}"
echo -e "  Config:  $CONFIG_DIR"
echo -e "  Data:    $DATA_DIR"
echo -e "  Logs:    $LOG_DIR"
echo -e "  Binary:  $INSTALL_DIR/watchflare-agent"
echo
