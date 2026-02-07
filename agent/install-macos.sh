#!/bin/bash
set -e

# Watchflare Agent - macOS Installation Script
# This script installs the Watchflare agent as a system service

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="io.watchflare.agent"
AGENT_USER="watchflare"
AGENT_GROUP="staff"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/watchflare"
DATA_DIR="/var/lib/watchflare"
PLIST_PATH="/Library/LaunchDaemons/${SERVICE_NAME}.plist"

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

# Step 1: Check if binary exists
if [ ! -f "./watchflare-agent" ]; then
    echo -e "${RED}Error: watchflare-agent binary not found in current directory${NC}"
    echo "Please build the agent first: go build -o watchflare-agent"
    exit 1
fi

echo -e "${YELLOW}[1/7]${NC} Checking for existing installation..."
if [ -f "$PLIST_PATH" ]; then
    echo "  → Found existing installation, stopping service..."
    launchctl bootout system/$SERVICE_NAME 2>/dev/null || true
    sleep 1
fi

# Step 2: Create system user
echo -e "${YELLOW}[2/7]${NC} Creating system user '${AGENT_USER}'..."

# Check if user already exists
if dscl . -read /Users/${AGENT_USER} >/dev/null 2>&1; then
    echo "  → User '${AGENT_USER}' already exists"
else
    # Find next available UID in system range (200-400 on macOS)
    MAX_UID=$(dscl . -list /Users UniqueID | awk '{print $2}' | sort -n | tail -1)
    NEXT_UID=$((MAX_UID + 1))

    # Ensure UID is in system range
    if [ $NEXT_UID -lt 200 ]; then
        NEXT_UID=200
    fi

    # Create user
    dscl . -create /Users/${AGENT_USER}
    dscl . -create /Users/${AGENT_USER} UniqueID ${NEXT_UID}
    dscl . -create /Users/${AGENT_USER} PrimaryGroupID 20  # staff group
    dscl . -create /Users/${AGENT_USER} UserShell /usr/bin/false
    dscl . -create /Users/${AGENT_USER} RealName "Watchflare Agent"
    dscl . -create /Users/${AGENT_USER} NFSHomeDirectory /var/empty
    dscl . -create /Users/${AGENT_USER} Password '*'

    echo "  → Created user '${AGENT_USER}' (UID: ${NEXT_UID})"
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

mkdir -p "${DATA_DIR}/brew-cache"
chown ${AGENT_USER}:${AGENT_GROUP} "${DATA_DIR}/brew-cache"
chmod 750 "${DATA_DIR}/brew-cache"
echo "  → Created ${DATA_DIR}/brew-cache"

# Step 4: Install binary
echo -e "${YELLOW}[4/7]${NC} Installing binary..."
cp ./watchflare-agent "${INSTALL_DIR}/watchflare-agent"
chown root:wheel "${INSTALL_DIR}/watchflare-agent"
chmod 755 "${INSTALL_DIR}/watchflare-agent"
echo "  → Installed to ${INSTALL_DIR}/watchflare-agent"

# Step 4b: Create log file with proper permissions
LOG_FILE="/var/log/watchflare-agent.log"
touch "$LOG_FILE"
chown ${AGENT_USER}:${AGENT_GROUP} "$LOG_FILE"
chmod 644 "$LOG_FILE"
echo "  → Created log file ${LOG_FILE}"

# Step 5: Create LaunchDaemon plist
echo -e "${YELLOW}[5/7]${NC} Creating LaunchDaemon..."
cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${SERVICE_NAME}</string>

    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/watchflare-agent</string>
    </array>

    <key>UserName</key>
    <string>${AGENT_USER}</string>

    <key>GroupName</key>
    <string>${AGENT_GROUP}</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>StandardOutPath</key>
    <string>/var/log/watchflare-agent.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/watchflare-agent.log</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>WATCHFLARE_CONFIG_DIR</key>
        <string>${CONFIG_DIR}</string>
        <key>WATCHFLARE_DATA_DIR</key>
        <string>${DATA_DIR}</string>
        <key>HOMEBREW_NO_AUTO_UPDATE</key>
        <string>1</string>
        <key>HOMEBREW_NO_INSTALL_CLEANUP</key>
        <string>1</string>
        <key>HOMEBREW_CACHE</key>
        <string>${DATA_DIR}/brew-cache</string>
    </dict>

    <key>ThrottleInterval</key>
    <integer>5</integer>
</dict>
</plist>
EOF

chown root:wheel "$PLIST_PATH"
chmod 644 "$PLIST_PATH"
echo "  → Created $PLIST_PATH"

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

# Step 7: Load service
if [ "$NEEDS_REGISTRATION" = false ]; then
    echo -e "${YELLOW}[7/7]${NC} Starting service..."
    launchctl bootstrap system "$PLIST_PATH"
    sleep 2

    # Check if service is running
    if launchctl print system/${SERVICE_NAME} >/dev/null 2>&1; then
        echo -e "  → ${GREEN}Service started successfully${NC}"
    else
        echo -e "  → ${RED}Service failed to start${NC}"
        echo "  → Check logs: tail -f /var/log/watchflare-agent.log"
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
echo "  LaunchDaemon:  ${PLIST_PATH}"
echo "  Logs:          /var/log/watchflare-agent.log"
echo ""

if [ "$NEEDS_REGISTRATION" = true ]; then
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Register the agent:"
    echo "     sudo ${INSTALL_DIR}/watchflare-agent register --token=YOUR_TOKEN --host=YOUR_HOST"
    echo ""
    echo "  2. Start the service:"
    echo "     sudo launchctl bootstrap system ${PLIST_PATH}"
    echo ""
else
    if [ -n "$TOKEN" ]; then
        echo "Registration details:"
        echo "  Backend: ${HOST}:${PORT}"
        echo ""
    fi
    echo "Service management:"
    echo "  Status:  sudo launchctl print system/${SERVICE_NAME}"
    echo "  Stop:    sudo launchctl bootout system/${SERVICE_NAME}"
    echo "  Start:   sudo launchctl bootstrap system ${PLIST_PATH}"
    echo "  Logs:    tail -f /var/log/watchflare-agent.log"
    echo ""
fi

echo -e "${GREEN}Installation successful!${NC}"
