#!/bin/bash
# Watchflare Agent Uninstallation Script for macOS

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
WATCHFLARE_USER="_watchflare"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/watchflare"
DATA_DIR="/var/lib/watchflare"
LOG_DIR="/var/log/watchflare"
PLIST_FILE="/Library/LaunchDaemons/com.watchflare.agent.plist"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}   Watchflare Agent - Uninstallation (macOS)${NC}"
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

# Step 1: Stop and unload service
echo -e "${BLUE}[1/6]${NC} Stopping service..."
if launchctl list | grep -q "com.watchflare.agent"; then
    launchctl unload "$PLIST_FILE" 2>/dev/null || true
    echo -e "${GREEN}✓${NC} Service stopped and unloaded"
else
    echo -e "${YELLOW}ℹ${NC}  Service not running"
fi

# Step 2: Remove plist file
echo
echo -e "${BLUE}[2/6]${NC} Removing launchd service..."
if [ -f "$PLIST_FILE" ]; then
    rm -f "$PLIST_FILE"
    echo -e "${GREEN}✓${NC} Service file removed"
fi

# Step 3: Remove binary
echo
echo -e "${BLUE}[3/6]${NC} Removing binary..."
if [ -f "$INSTALL_DIR/watchflare-agent" ]; then
    rm -f "$INSTALL_DIR/watchflare-agent"
    echo -e "${GREEN}✓${NC} Binary removed"
fi

# Step 4: Remove newsyslog config
echo
echo -e "${BLUE}[4/6]${NC} Removing log rotation configuration..."
if [ -f "/etc/newsyslog.d/watchflare.conf" ]; then
    rm -f /etc/newsyslog.d/watchflare.conf
    echo -e "${GREEN}✓${NC} Log rotation config removed"
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
if dscl . -read /Users/$WATCHFLARE_USER &>/dev/null; then
    dscl . -delete /Users/$WATCHFLARE_USER
    echo -e "${GREEN}✓${NC} User removed"
fi

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ Uninstallation completed${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
