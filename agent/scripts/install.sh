#!/bin/bash
set -e

# Watchflare Agent - Bootstrap Installation Script
# This script downloads and installs the Watchflare agent

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="Kilian-Pichard/watchflare"
BINARY_NAME="watchflare-agent"

# Check if --local flag is present
LOCAL_MODE=false
if [ "$1" = "--local" ]; then
    LOCAL_MODE=true
    shift  # Remove --local from arguments
fi

echo -e "${GREEN}=== Watchflare Agent Installation ===${NC}"
echo ""

# Step 1: Detect OS and architecture
echo "Detecting system..."
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    arm64)
        # Keep as is (macOS)
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        echo "Supported architectures: x86_64 (amd64), aarch64/arm64"
        exit 1
        ;;
esac

# Normalize OS names
case "$OS" in
    linux)
        ;;
    darwin)
        ;;
    *)
        echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
        echo "Supported systems: Linux, macOS (Darwin)"
        exit 1
        ;;
esac

PLATFORM="${OS}_${ARCH}"
echo -e "  → Detected: ${YELLOW}${OS}${NC} on ${YELLOW}${ARCH}${NC} (${PLATFORM})"

# Step 2: Determine binary source
if [ "$LOCAL_MODE" = true ]; then
    # Local mode: use dist/ directory
    echo ""
    echo "Running in LOCAL MODE (development)"

    # Find the agent directory (current dir or parent)
    if [ -d "./dist/${PLATFORM}" ]; then
        BINARY_PATH="./dist/${PLATFORM}/${BINARY_NAME}"
    elif [ -d "../dist/${PLATFORM}" ]; then
        BINARY_PATH="../dist/${PLATFORM}/${BINARY_NAME}"
    else
        echo -e "${RED}Error: Binary not found in ./dist/${PLATFORM}/${BINARY_NAME}${NC}"
        echo ""
        echo "Build the agent first:"
        echo "  cd agent/"
        echo "  ./build-all.sh"
        exit 1
    fi

    if [ ! -f "$BINARY_PATH" ]; then
        echo -e "${RED}Error: Binary not found: $BINARY_PATH${NC}"
        exit 1
    fi

    echo "  → Using local binary: ${BINARY_PATH}"

    # Copy to /tmp for installation
    TMP_BINARY="/tmp/${BINARY_NAME}"
    cp "$BINARY_PATH" "$TMP_BINARY"
    chmod +x "$TMP_BINARY"
else
    # Production mode: download from GitHub releases
    echo ""
    echo "Downloading agent from GitHub releases..."

    # Get latest version tag
    VERSION=$(curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//')
    if [ -z "$VERSION" ]; then
        echo -e "${RED}Error: Failed to detect latest version${NC}"
        exit 1
    fi
    echo -e "  → Latest version: ${YELLOW}${VERSION}${NC}"

    ARCHIVE_NAME="${BINARY_NAME}_${VERSION}_${PLATFORM}.tar.gz"
    ARCHIVE_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${ARCHIVE_NAME}"
    TMP_ARCHIVE="/tmp/${ARCHIVE_NAME}"
    TMP_BINARY="/tmp/${BINARY_NAME}"

    # Download archive
    if command -v curl >/dev/null 2>&1; then
        echo "  → Downloading ${ARCHIVE_URL}"
        if ! curl -fsSL "$ARCHIVE_URL" -o "$TMP_ARCHIVE"; then
            echo -e "${RED}Error: Failed to download archive${NC}"
            echo "URL: $ARCHIVE_URL"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        echo "  → Downloading ${ARCHIVE_URL}"
        if ! wget -q "$ARCHIVE_URL" -O "$TMP_ARCHIVE"; then
            echo -e "${RED}Error: Failed to download archive${NC}"
            echo "URL: $ARCHIVE_URL"
            exit 1
        fi
    else
        echo -e "${RED}Error: Neither curl nor wget found${NC}"
        echo "Please install curl or wget and try again"
        exit 1
    fi

    # Extract binary
    tar xzf "$TMP_ARCHIVE" -C /tmp
    rm -f "$TMP_ARCHIVE"
    chmod +x "$TMP_BINARY"
    echo -e "  → ${GREEN}Downloaded and extracted successfully${NC}"
fi

# Step 3: Run installation
echo ""
echo "Starting installation..."
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: Installation requires root privileges${NC}"
    echo "Running with sudo..."
    echo ""

    # Re-run with sudo, passing all arguments
    exec sudo "$TMP_BINARY" install "$@"
else
    # Already root, run directly
    exec "$TMP_BINARY" install "$@"
fi
