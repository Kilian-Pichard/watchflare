#!/bin/bash
# Watchflare Agent - Universal Installation Script
# Automatically detects OS and runs the appropriate installer
# Usage: curl -sSL https://get.watchflare.io/ | sudo bash -s -- --token TOKEN --host HOST

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
GITHUB_REPO="watchflare/watchflare"
BACKEND_HOST=""
BACKEND_PORT="50051"
REGISTRATION_TOKEN=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --token)
            REGISTRATION_TOKEN="$2"
            shift 2
            ;;
        --host)
            BACKEND_HOST="$2"
            shift 2
            ;;
        --port)
            BACKEND_PORT="$2"
            shift 2
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   Watchflare Agent - Automated Installation${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Validate required parameters
if [ -z "$REGISTRATION_TOKEN" ]; then
    echo -e "${RED}✗ Registration token is required${NC}"
    echo -e "${YELLOW}Usage: curl -sSL https://get.watchflare.io/ | sudo bash -s -- --token TOKEN --host HOST${NC}"
    exit 1
fi

if [ -z "$BACKEND_HOST" ]; then
    echo -e "${RED}✗ Backend host is required${NC}"
    echo -e "${YELLOW}Usage: curl -sSL https://get.watchflare.io/ | sudo bash -s -- --token TOKEN --host HOST${NC}"
    exit 1
fi

# Detect OS
echo -e "${BLUE}[1/4]${NC} Detecting operating system..."
OS="unknown"
ARCH=$(uname -m)

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        DISTRO=$ID
        VERSION=$VERSION_ID
        echo -e "${GREEN}✓${NC} Detected: $PRETTY_NAME"
    else
        echo -e "${YELLOW}ℹ${NC}  Detected: Linux (unknown distribution)"
    fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="darwin"
    MACOS_VERSION=$(sw_vers -productVersion)
    echo -e "${GREEN}✓${NC} Detected: macOS $MACOS_VERSION"
else
    echo -e "${RED}✗ Unsupported operating system: $OSTYPE${NC}"
    exit 1
fi

# Normalize architecture
case $ARCH in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}✗ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}Architecture:${NC} $ARCH"

# Get binary (local or download)
echo
echo -e "${BLUE}[2/4]${NC} Getting Watchflare Agent binary..."

BINARY_NAME="watchflare-agent-${OS}-${ARCH}"
USED_LOCAL_BINARY=false

# Check if local binary exists (for development/testing)
if [ -f "./watchflare-agent" ]; then
    echo -e "${YELLOW}ℹ${NC}  Using local binary for testing"
    chmod +x watchflare-agent
    USED_LOCAL_BINARY=true
elif [ -f "../watchflare-agent" ]; then
    echo -e "${YELLOW}ℹ${NC}  Using local binary from parent directory"
    cp ../watchflare-agent ./watchflare-agent
    chmod +x watchflare-agent
    USED_LOCAL_BINARY=true
else
    # Download from GitHub releases
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BINARY_NAME}"
    echo -e "${BLUE}Downloading from:${NC} $DOWNLOAD_URL"

    if command -v curl &> /dev/null; then
        curl -L -o watchflare-agent "$DOWNLOAD_URL"
    elif command -v wget &> /dev/null; then
        wget -O watchflare-agent "$DOWNLOAD_URL"
    else
        echo -e "${RED}✗ Neither curl nor wget found. Please install one of them.${NC}"
        exit 1
    fi

    chmod +x watchflare-agent
fi

echo -e "${GREEN}✓${NC} Binary ready"

# Get and run installer script
echo
echo -e "${BLUE}[3/4]${NC} Running installation script..."

if [ "$OS" = "linux" ]; then
    INSTALLER_SCRIPT="install.sh"
elif [ "$OS" = "darwin" ]; then
    INSTALLER_SCRIPT="install-macos.sh"
fi

# Check if local installer exists (for development/testing)
if [ -f "./$INSTALLER_SCRIPT" ]; then
    echo -e "${YELLOW}ℹ${NC}  Using local installer script"
    ./$INSTALLER_SCRIPT
else
    # Download from GitHub
    INSTALLER_URL="https://raw.githubusercontent.com/${GITHUB_REPO}/main/agent/$INSTALLER_SCRIPT"
    echo -e "${BLUE}Downloading installer from:${NC} $INSTALLER_URL"
    curl -sSL "$INSTALLER_URL" -o install-platform.sh
    chmod +x install-platform.sh
    ./install-platform.sh
    rm -f install-platform.sh
fi

# Clean up downloaded binary (but keep local binary for development)
if [ "$USED_LOCAL_BINARY" = false ]; then
    rm -f watchflare-agent
fi

# Register agent
echo
echo -e "${BLUE}[4/4]${NC} Registering agent..."

AGENT_CMD="/usr/local/bin/watchflare-agent"
if [ "$OS" = "linux" ]; then
    AGENT_CMD="/usr/bin/watchflare-agent"
fi

$AGENT_CMD --token "$REGISTRATION_TOKEN" --host "$BACKEND_HOST" --port "$BACKEND_PORT" --register-only

echo -e "${GREEN}✓${NC} Agent registered successfully"

# Start service
echo
echo -e "${BLUE}Starting service...${NC}"

if [ "$OS" = "linux" ]; then
    systemctl start watchflare-agent
    systemctl enable watchflare-agent
    echo -e "${GREEN}✓${NC} Service started"
    echo
    echo -e "${YELLOW}Check status:${NC} sudo systemctl status watchflare-agent"
    echo -e "${YELLOW}View logs:${NC} sudo journalctl -u watchflare-agent -f"
elif [ "$OS" = "darwin" ]; then
    launchctl load /Library/LaunchDaemons/com.watchflare.agent.plist
    echo -e "${GREEN}✓${NC} Service started"
    echo
    echo -e "${YELLOW}Check status:${NC} sudo launchctl list | grep watchflare"
    echo -e "${YELLOW}View logs:${NC} tail -f /var/log/watchflare/watchflare-agent.log"
fi

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ Installation completed successfully!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
echo -e "${YELLOW}Your server is now being monitored!${NC}"
echo -e "Visit your dashboard to see real-time status updates."
echo
