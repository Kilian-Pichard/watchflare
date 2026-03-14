#!/bin/bash
set -e

# Watchflare Agent - Linux Uninstallation Script
# This script removes the Watchflare agent and all its files

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="watchflare-agent"
AGENT_USER="watchflare"
AGENT_GROUP="watchflare"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/watchflare"
DATA_DIR="/var/lib/watchflare"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
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

# Step 1: Stop and disable service
echo -e "${YELLOW}[1/7]${NC} Stopping service..."
if systemctl is-active --quiet ${SERVICE_NAME}; then
    systemctl stop ${SERVICE_NAME}
    echo "  → Service stopped"
else
    echo "  → Service not running"
fi

if systemctl is-enabled --quiet ${SERVICE_NAME} 2>/dev/null; then
    systemctl disable ${SERVICE_NAME}
    echo "  → Service disabled"
fi

# Step 2: Remove systemd service file
echo -e "${YELLOW}[2/7]${NC} Removing systemd service..."
if [ -f "$SERVICE_FILE" ]; then
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    echo "  → Removed $SERVICE_FILE"
    echo "  → Systemd daemon reloaded"
else
    echo "  → Service file not found"
fi

# Step 3: Remove binary
echo -e "${YELLOW}[3/7]${NC} Removing binary..."
if [ -f "${INSTALL_DIR}/watchflare-agent" ]; then
    rm -f "${INSTALL_DIR}/watchflare-agent"
    echo "  → Removed ${INSTALL_DIR}/watchflare-agent"
else
    echo "  → Binary not found"
fi

# Step 4: Remove data directory
echo -e "${YELLOW}[4/7]${NC} Removing data directory..."
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

# Step 5: Remove agent configuration directory
echo -e "${YELLOW}[5/7]${NC} Removing agent configuration..."
if [ -d "$CONFIG_DIR" ]; then
    # Ask for confirmation before removing config directory
    read -p "Remove configuration directory ${CONFIG_DIR}? (contains agent.conf, ca.pem) [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
        echo "  → Removed $CONFIG_DIR"
    else
        echo "  → Kept $CONFIG_DIR"
    fi
else
    echo "  → Configuration directory not found"
fi

# Step 6: Remove system user and group
echo -e "${YELLOW}[6/7]${NC} Removing system user..."
if id -u ${AGENT_USER} >/dev/null 2>&1; then
    read -p "Remove system user '${AGENT_USER}'? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        userdel ${AGENT_USER} 2>/dev/null || true
        echo "  → Removed user '${AGENT_USER}'"

        # Remove group if it exists and is empty
        if getent group ${AGENT_GROUP} >/dev/null 2>&1; then
            groupdel ${AGENT_GROUP} 2>/dev/null || true
            echo "  → Removed group '${AGENT_GROUP}'"
        fi
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
if [ -d "$CONFIG_DIR" ]; then
    echo "  - Configuration directory: ${CONFIG_DIR}/"
fi
if [ -d "$DATA_DIR" ]; then
    echo "  - Data directory: ${DATA_DIR}/"
fi
if id -u ${AGENT_USER} >/dev/null 2>&1; then
    echo "  - User: ${AGENT_USER}"
fi
if [ -f "$LOG_FILE" ]; then
    echo "  - Logs: ${LOG_FILE}"
fi
echo ""
echo -e "${GREEN}Uninstallation successful!${NC}"
