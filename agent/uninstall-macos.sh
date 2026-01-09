#!/bin/bash
set -e

# Watchflare Agent - macOS Uninstallation Script
# This script removes the Watchflare agent and all its files

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="com.watchflare.agent"
AGENT_USER="watchflare"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/watchflare"
DATA_DIR="/var/lib/watchflare"
PLIST_PATH="/Library/LaunchDaemons/${SERVICE_NAME}.plist"
LOG_FILE="/var/log/watchflare-agent.log"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (use sudo)${NC}"
    exit 1
fi

echo -e "${YELLOW}=== Watchflare Agent Uninstallation ===${NC}\n"

# Confirmation prompt
read -p "This will remove the Watchflare agent and all its data. Continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Uninstallation cancelled."
    exit 0
fi

echo ""

# Step 1: Stop and unload service
echo -e "${YELLOW}[1/6]${NC} Stopping service..."
if [ -f "$PLIST_PATH" ]; then
    if launchctl print system/${SERVICE_NAME} >/dev/null 2>&1; then
        launchctl bootout system/${SERVICE_NAME} 2>/dev/null || true
        echo "  → Service stopped"
    else
        echo "  → Service not running"
    fi
else
    echo "  → Service not installed"
fi

# Step 2: Remove LaunchDaemon plist
echo -e "${YELLOW}[2/6]${NC} Removing LaunchDaemon..."
if [ -f "$PLIST_PATH" ]; then
    rm -f "$PLIST_PATH"
    echo "  → Removed $PLIST_PATH"
else
    echo "  → LaunchDaemon not found"
fi

# Step 3: Remove binary
echo -e "${YELLOW}[3/6]${NC} Removing binary..."
if [ -f "${INSTALL_DIR}/watchflare-agent" ]; then
    rm -f "${INSTALL_DIR}/watchflare-agent"
    echo "  → Removed ${INSTALL_DIR}/watchflare-agent"
else
    echo "  → Binary not found"
fi

# Step 4: Remove data directory
echo -e "${YELLOW}[4/6]${NC} Removing data directory..."
if [ -d "$DATA_DIR" ]; then
    # Ask for confirmation before removing data
    read -p "Remove data directory ${DATA_DIR}? (contains WAL and package state) [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$DATA_DIR"
        echo "  → Removed $DATA_DIR"
    else
        echo "  → Kept $DATA_DIR"
    fi
else
    echo "  → Data directory not found"
fi

# Step 5: Remove agent configuration (preserving backend PKI)
echo -e "${YELLOW}[5/6]${NC} Removing agent configuration..."
if [ -f "${CONFIG_DIR}/agent.conf" ]; then
    # Ask for confirmation before removing config
    read -p "Remove agent configuration ${CONFIG_DIR}/agent.conf? (contains agent credentials) [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -f "${CONFIG_DIR}/agent.conf"
        echo "  → Removed ${CONFIG_DIR}/agent.conf"
        echo "  → Preserved ${CONFIG_DIR}/pki/ (backend certificates)"
    else
        echo "  → Kept ${CONFIG_DIR}/agent.conf"
    fi
else
    echo "  → Agent configuration not found"
    if [ -d "${CONFIG_DIR}/pki" ]; then
        echo "  → Backend PKI directory preserved"
    fi
fi

# Step 6: Remove system user
echo -e "${YELLOW}[6/6]${NC} Removing system user..."
if dscl . -read /Users/${AGENT_USER} >/dev/null 2>&1; then
    read -p "Remove system user '${AGENT_USER}'? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        dscl . -delete /Users/${AGENT_USER}
        echo "  → Removed user '${AGENT_USER}'"
    else
        echo "  → Kept user '${AGENT_USER}'"
    fi
else
    echo "  → User '${AGENT_USER}' not found"
fi

# Step 7: Remove log file
echo -e "${YELLOW}[7/7]${NC} Removing log file..."
if [ -f "$LOG_FILE" ]; then
    read -p "Remove log file ${LOG_FILE}? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -f "$LOG_FILE"
        echo "  → Removed $LOG_FILE"
    else
        echo "  → Kept $LOG_FILE"
    fi
else
    echo "  → Log file not found"
fi

# Summary
echo ""
echo -e "${GREEN}=== Uninstallation Complete ===${NC}"
echo ""
echo "The following items may still exist:"
if [ -f "${CONFIG_DIR}/agent.conf" ]; then
    echo "  - Agent configuration: ${CONFIG_DIR}/agent.conf"
fi
if [ -d "${CONFIG_DIR}/pki" ]; then
    echo "  - Backend PKI (preserved): ${CONFIG_DIR}/pki/"
fi
if [ -d "$DATA_DIR" ]; then
    echo "  - Data: ${DATA_DIR}/"
fi
if dscl . -read /Users/${AGENT_USER} >/dev/null 2>&1; then
    echo "  - User: ${AGENT_USER}"
fi
if [ -f "$LOG_FILE" ]; then
    echo "  - Logs: ${LOG_FILE}"
fi
echo ""
echo -e "${GREEN}Uninstallation successful!${NC}"
