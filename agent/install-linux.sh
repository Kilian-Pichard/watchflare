#!/bin/bash
set -e

# Watchflare Agent - Linux Installation Script
# This script installs the Watchflare agent as a systemd service

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

# Parse command line arguments
TOKEN=""
HOST=""
PORT=""
for arg in "$@"; do
    case $arg in
        --token=*)
            TOKEN="${arg#*=}"
            ;;
        --host=*)
            HOST="${arg#*=}"
            ;;
        --port=*)
            PORT="${arg#*=}"
            ;;
        *)
            echo -e "${RED}Unknown argument: $arg${NC}"
            echo "Usage: sudo $0 [--token=TOKEN] [--host=HOST] [--port=PORT]"
            exit 1
            ;;
    esac
done

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (use sudo)${NC}"
    exit 1
fi

echo -e "${GREEN}=== Watchflare Agent Installation ===${NC}\n"

# Step 1: Detect architecture and find binary
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        BINARY_NAME="watchflare-agent-linux-amd64"
        ;;
    aarch64|arm64)
        BINARY_NAME="watchflare-agent-linux-arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        echo "Supported: x86_64 (amd64), aarch64/arm64"
        exit 1
        ;;
esac

# Check if specific binary exists, otherwise fallback to generic name
if [ -f "./$BINARY_NAME" ]; then
    AGENT_BINARY="./$BINARY_NAME"
    echo "  → Detected architecture: $ARCH (using $BINARY_NAME)"
elif [ -f "./watchflare-agent" ]; then
    AGENT_BINARY="./watchflare-agent"
    echo "  → Using generic binary: watchflare-agent"
else
    echo -e "${RED}Error: Agent binary not found${NC}"
    echo "Expected: ./$BINARY_NAME or ./watchflare-agent"
    echo ""
    echo "Build the agent:"
    echo "  - Single platform: GOOS=linux GOARCH=amd64 go build -o watchflare-agent"
    echo "  - All platforms: ./build-all.sh"
    exit 1
fi

echo -e "${YELLOW}[1/7]${NC} Checking for existing installation..."

# Detect if systemd is available
HAS_SYSTEMD=false
if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
    HAS_SYSTEMD=true
    echo "  → Systemd detected"

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        echo "  → Found existing installation, stopping service..."
        systemctl stop ${SERVICE_NAME}
        sleep 1
    fi
else
    echo "  → Systemd not available (container environment detected)"
    echo "  → Will install without systemd service"
fi

# Step 2: Create system user and group
echo -e "${YELLOW}[2/7]${NC} Creating system user '${AGENT_USER}'..."

# Check if group exists
if ! getent group ${AGENT_GROUP} >/dev/null 2>&1; then
    groupadd --system ${AGENT_GROUP}
    echo "  → Created group '${AGENT_GROUP}'"
else
    echo "  → Group '${AGENT_GROUP}' already exists"
fi

# Check if user exists
if ! id -u ${AGENT_USER} >/dev/null 2>&1; then
    useradd --system \
        --gid ${AGENT_GROUP} \
        --home-dir /var/empty \
        --shell /usr/sbin/nologin \
        --comment "Watchflare Agent" \
        ${AGENT_USER}
    echo "  → Created user '${AGENT_USER}'"
else
    echo "  → User '${AGENT_USER}' already exists"
fi

# Step 3: Create directories
echo -e "${YELLOW}[3/7]${NC} Creating directories..."

mkdir -p "$CONFIG_DIR"
chown root:${AGENT_GROUP} "$CONFIG_DIR"
chmod 750 "$CONFIG_DIR"
echo "  → Created $CONFIG_DIR"

mkdir -p "$DATA_DIR"
chown ${AGENT_USER}:${AGENT_GROUP} "$DATA_DIR"
chmod 750 "$DATA_DIR"
echo "  → Created $DATA_DIR"

mkdir -p "${DATA_DIR}/wal"
chown ${AGENT_USER}:${AGENT_GROUP} "${DATA_DIR}/wal"
chmod 750 "${DATA_DIR}/wal"
echo "  → Created ${DATA_DIR}/wal"

# Step 4: Install binary
echo -e "${YELLOW}[4/7]${NC} Installing binary..."
cp "$AGENT_BINARY" "${INSTALL_DIR}/watchflare-agent"
chown root:root "${INSTALL_DIR}/watchflare-agent"
chmod 755 "${INSTALL_DIR}/watchflare-agent"
echo "  → Installed to ${INSTALL_DIR}/watchflare-agent"

# Create log file with proper permissions
touch "$LOG_FILE"
chown ${AGENT_USER}:${AGENT_GROUP} "$LOG_FILE"
chmod 644 "$LOG_FILE"
echo "  → Created log file ${LOG_FILE}"

# Step 5: Install systemd service (if available)
echo -e "${YELLOW}[5/7]${NC} Installing systemd service..."

if [ "$HAS_SYSTEMD" = true ]; then
    # Check if service file exists in current directory
    if [ ! -f "./watchflare-agent.service" ]; then
        echo -e "${RED}Error: watchflare-agent.service file not found${NC}"
        exit 1
    fi

    cp ./watchflare-agent.service "$SERVICE_FILE"
    chown root:root "$SERVICE_FILE"
    chmod 644 "$SERVICE_FILE"
    echo "  → Installed to $SERVICE_FILE"

    # Reload systemd
    systemctl daemon-reload
    echo "  → Systemd daemon reloaded"
else
    echo "  → Skipped (systemd not available)"
fi

# Step 6: Registration
echo -e "${YELLOW}[6/7]${NC} Agent registration..."
if [ ! -f "${CONFIG_DIR}/agent.conf" ]; then
    # No existing config, check if we have registration parameters
    if [ -n "$TOKEN" ]; then
        echo "  → Registering agent with backend..."

        # Set defaults if not provided
        if [ -z "$HOST" ]; then
            HOST="localhost"
        fi
        if [ -z "$PORT" ]; then
            PORT="50051"
        fi

        # Run registration
        if "${INSTALL_DIR}/watchflare-agent" register --token="$TOKEN" --host="$HOST" --port="$PORT"; then
            echo -e "  → ${GREEN}Registration successful${NC}"
            NEEDS_REGISTRATION=false
        else
            echo -e "  → ${RED}Registration failed${NC}"
            NEEDS_REGISTRATION=true
        fi
    else
        echo -e "${YELLOW}  ⚠ No configuration file found${NC}"
        echo "  → To register now, run:"
        echo "     sudo ${INSTALL_DIR}/watchflare-agent register --token=YOUR_TOKEN --host=YOUR_HOST"
        NEEDS_REGISTRATION=true
    fi
else
    echo "  → Configuration file already exists"
    NEEDS_REGISTRATION=false
fi

# Step 7: Enable and start service
if [ "$NEEDS_REGISTRATION" = false ]; then
    echo -e "${YELLOW}[7/7]${NC} Starting service..."

    if [ "$HAS_SYSTEMD" = true ]; then
        # Enable service
        systemctl enable ${SERVICE_NAME}
        echo "  → Service enabled (will start on boot)"

        # Start service
        systemctl start ${SERVICE_NAME}
        sleep 2

        # Check if service is running
        if systemctl is-active --quiet ${SERVICE_NAME}; then
            echo -e "  → ${GREEN}Service started successfully${NC}"
        else
            echo -e "  → ${RED}Service failed to start${NC}"
            echo "  → Check logs: journalctl -u ${SERVICE_NAME} -f"
            echo "  → Or: tail -f ${LOG_FILE}"
        fi
    else
        echo -e "${YELLOW}  → Cannot start service (systemd not available)${NC}"
        echo "  → To start the agent manually:"
        echo "     ${INSTALL_DIR}/watchflare-agent"
        echo "  → Or run in background:"
        echo "     nohup ${INSTALL_DIR}/watchflare-agent > ${LOG_FILE} 2>&1 &"
    fi
else
    echo -e "${YELLOW}[7/7]${NC} Skipping service start (needs registration)"
fi

# Summary
echo ""
echo -e "${GREEN}=== Installation Complete ===${NC}"
echo ""
echo "Installation paths:"
echo "  Binary:        ${INSTALL_DIR}/watchflare-agent"
echo "  Configuration: ${CONFIG_DIR}/"
echo "  Data:          ${DATA_DIR}/"
echo "  Service:       ${SERVICE_FILE}"
echo "  Logs:          ${LOG_FILE}"
echo ""

if [ "$NEEDS_REGISTRATION" = true ]; then
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Register the agent:"
    echo "     sudo ${INSTALL_DIR}/watchflare-agent register --token=YOUR_TOKEN --host=YOUR_HOST"
    echo ""
    if [ "$HAS_SYSTEMD" = true ]; then
        echo "  2. Start the service:"
        echo "     sudo systemctl enable ${SERVICE_NAME}"
        echo "     sudo systemctl start ${SERVICE_NAME}"
    else
        echo "  2. Start the agent:"
        echo "     ${INSTALL_DIR}/watchflare-agent"
        echo "     Or in background: nohup ${INSTALL_DIR}/watchflare-agent > ${LOG_FILE} 2>&1 &"
    fi
    echo ""
else
    if [ -n "$TOKEN" ]; then
        echo "Registration details:"
        echo "  Backend: ${HOST}:${PORT}"
        echo ""
    fi

    if [ "$HAS_SYSTEMD" = true ]; then
        echo "Service management:"
        echo "  Status:  sudo systemctl status ${SERVICE_NAME}"
        echo "  Stop:    sudo systemctl stop ${SERVICE_NAME}"
        echo "  Start:   sudo systemctl start ${SERVICE_NAME}"
        echo "  Restart: sudo systemctl restart ${SERVICE_NAME}"
        echo "  Logs:    journalctl -u ${SERVICE_NAME} -f"
        echo "  Or:      tail -f ${LOG_FILE}"
    else
        echo "Agent management (no systemd):"
        echo "  Start:   ${INSTALL_DIR}/watchflare-agent"
        echo "  Background: nohup ${INSTALL_DIR}/watchflare-agent > ${LOG_FILE} 2>&1 &"
        echo "  Logs:    tail -f ${LOG_FILE}"
        echo "  Stop:    pkill -f watchflare-agent"
    fi
    echo ""
fi

echo -e "${GREEN}Installation successful!${NC}"
