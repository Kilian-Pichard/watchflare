#!/bin/bash
# Watchflare Agent Uninstallation Script

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
WATCHFLARE_USER="watchflare"
INSTALL_DIR="/usr/bin"
CONFIG_DIR="/etc/watchflare"
DATA_DIR="/var/lib/watchflare"
LOG_DIR="/var/log/watchflare"
SERVICE_FILE="/etc/systemd/system/watchflare-agent.service"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}   Watchflare Agent - Uninstallation${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}✗ This script must be run as root${NC}"
    exit 1
fi

# Ask for confirmation
echo -e "${YELLOW}⚠ Warning: This will remove Watchflare Agent and all its data${NC}"
read -p "Are you sure you want to continue? (yes/NO): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo -e "${BLUE}Uninstallation cancelled${NC}"
    exit 0
fi

# Step 1: Stop and disable service
echo -e "${BLUE}[1/6]${NC} Stopping service..."
if systemctl is-active --quiet watchflare-agent; then
    systemctl stop watchflare-agent
    echo -e "${GREEN}✓${NC} Service stopped"
else
    echo -e "${YELLOW}ℹ${NC}  Service not running"
fi

if systemctl is-enabled --quiet watchflare-agent 2>/dev/null; then
    systemctl disable watchflare-agent
    echo -e "${GREEN}✓${NC} Service disabled"
fi

# Step 2: Remove service file
echo
echo -e "${BLUE}[2/6]${NC} Removing systemd service..."
if [ -f "$SERVICE_FILE" ]; then
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    echo -e "${GREEN}✓${NC} Service file removed"
fi

# Step 3: Remove binary
echo
echo -e "${BLUE}[3/6]${NC} Removing binary..."
if [ -f "$INSTALL_DIR/watchflare-agent" ]; then
    rm -f "$INSTALL_DIR/watchflare-agent"
    echo -e "${GREEN}✓${NC} Binary removed"
fi

# Step 4: Remove logrotate config
echo
echo -e "${BLUE}[4/6]${NC} Removing logrotate configuration..."
if [ -f "/etc/logrotate.d/watchflare" ]; then
    rm -f /etc/logrotate.d/watchflare
    echo -e "${GREEN}✓${NC} Logrotate config removed"
fi

# Step 5: Remove directories (ask first)
echo
echo -e "${BLUE}[5/6]${NC} Removing directories..."
read -p "Remove configuration and data directories? This will delete all logs and config (yes/NO): " -r
if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    rm -rf "$CONFIG_DIR"
    rm -rf "$DATA_DIR"
    rm -rf "$LOG_DIR"
    echo -e "${GREEN}✓${NC} Directories removed"
else
    echo -e "${YELLOW}ℹ${NC}  Directories preserved"
    echo -e "  Config: $CONFIG_DIR"
    echo -e "  Data:   $DATA_DIR"
    echo -e "  Logs:   $LOG_DIR"
fi

# Step 6: Remove user
echo
echo -e "${BLUE}[6/6]${NC} Removing system user..."
if id "$WATCHFLARE_USER" &>/dev/null; then
    userdel "$WATCHFLARE_USER"
    echo -e "${GREEN}✓${NC} User removed"
fi

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ Uninstallation completed${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
