#!/bin/bash
# Watchflare Agent Installation Script for macOS
# Professional installation following macOS standards

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WATCHFLARE_USER="_watchflare"
WATCHFLARE_GROUP="staff"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/watchflare"
DATA_DIR="/var/lib/watchflare"
LOG_DIR="/var/log/watchflare"
PLIST_FILE="/Library/LaunchDaemons/com.watchflare.agent.plist"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   Watchflare Agent - macOS Installation${NC}"
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
echo -e "${BLUE}[1/6]${NC} Creating system user..."

# Check if user exists
if dscl . -read /Users/$WATCHFLARE_USER &>/dev/null; then
    echo -e "${YELLOW}ℹ${NC}  User '$WATCHFLARE_USER' already exists"
else
    # Find the next available UID in the system range (200-400)
    MAXID=$(dscl . -list /Users UniqueID | awk '{print $2}' | sort -n | tail -1)
    NEWID=$((MAXID + 1))

    # Ensure it's in the system range
    if [ $NEWID -lt 200 ]; then
        NEWID=200
    fi

    # Create the user
    dscl . -create /Users/$WATCHFLARE_USER
    dscl . -create /Users/$WATCHFLARE_USER UserShell /usr/bin/false
    dscl . -create /Users/$WATCHFLARE_USER RealName "Watchflare Agent"
    dscl . -create /Users/$WATCHFLARE_USER UniqueID $NEWID
    dscl . -create /Users/$WATCHFLARE_USER PrimaryGroupID 20  # staff group
    dscl . -create /Users/$WATCHFLARE_USER NFSHomeDirectory /var/empty
    dscl . -create /Users/$WATCHFLARE_USER Password '*'  # No password login

    echo -e "${GREEN}✓${NC} User '$WATCHFLARE_USER' created with UID $NEWID"
fi

# Step 2: Create directory structure
echo
echo -e "${BLUE}[2/6]${NC} Creating directory structure..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"/{logs,run,cache}
mkdir -p "$LOG_DIR"
echo -e "${GREEN}✓${NC} Directories created"

# Step 3: Set permissions
echo
echo -e "${BLUE}[3/6]${NC} Configuring permissions..."

# Config directory: root owns, watchflare can read
chown -R root:staff "$CONFIG_DIR"
chmod 750 "$CONFIG_DIR"

# Data directory: watchflare owns everything
chown -R $WATCHFLARE_USER:staff "$DATA_DIR"
chmod -R 750 "$DATA_DIR"
chmod 700 "$DATA_DIR/cache"  # Cache is more restrictive

# Log directory: watchflare owns
chown -R $WATCHFLARE_USER:staff "$LOG_DIR"
chmod 750 "$LOG_DIR"

echo -e "${GREEN}✓${NC} Permissions configured"

# Step 4: Install binary
echo
echo -e "${BLUE}[4/6]${NC} Installing binary..."
if [ -f "./watchflare-agent" ]; then
    cp ./watchflare-agent "$INSTALL_DIR/watchflare-agent"
    chmod 755 "$INSTALL_DIR/watchflare-agent"
    chown root:wheel "$INSTALL_DIR/watchflare-agent"
    echo -e "${GREEN}✓${NC} Binary installed to $INSTALL_DIR"
else
    echo -e "${RED}✗ Binary not found${NC}"
    echo -e "${YELLOW}  Please build the agent first: make build${NC}"
    exit 1
fi

# Step 5: Create launchd plist
echo
echo -e "${BLUE}[5/6]${NC} Installing launchd service..."
cat > "$PLIST_FILE" <<EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.watchflare.agent</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/watchflare-agent</string>
    </array>

    <key>UserName</key>
    <string>$WATCHFLARE_USER</string>

    <key>GroupName</key>
    <string>staff</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/var/log/watchflare/watchflare-agent.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/watchflare/watchflare-agent-error.log</string>

    <key>WorkingDirectory</key>
    <string>/var/lib/watchflare</string>

    <key>ThrottleInterval</key>
    <integer>5</integer>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
EOL

# Set proper permissions for plist
chmod 644 "$PLIST_FILE"
chown root:wheel "$PLIST_FILE"

echo -e "${GREEN}✓${NC} Launchd service created"

# Step 6: Install newsyslog configuration (log rotation for macOS)
echo
echo -e "${BLUE}[6/6]${NC} Installing log rotation configuration..."
cat > /etc/newsyslog.d/watchflare.conf <<EOL
# Watchflare Agent log rotation
# logfilename                                   [owner:group]  mode count size when  flags
/var/log/watchflare/watchflare-agent.log        $WATCHFLARE_USER:staff  640  7     1024 *     GZ
/var/log/watchflare/watchflare-agent-error.log  $WATCHFLARE_USER:staff  640  7     1024 *     GZ
EOL

echo -e "${GREEN}✓${NC} Log rotation configuration installed"

# Final summary
echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ Installation completed successfully!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Configure the agent with your registration token:"
echo -e "     ${BLUE}sudo watchflare-agent --token YOUR_TOKEN --host YOUR_BACKEND_HOST${NC}"
echo
echo -e "  2. Load and start the service:"
echo -e "     ${BLUE}sudo launchctl load $PLIST_FILE${NC}"
echo
echo -e "  3. Check status:"
echo -e "     ${BLUE}sudo launchctl list | grep watchflare${NC}"
echo
echo -e "  4. View logs:"
echo -e "     ${BLUE}tail -f /var/log/watchflare/watchflare-agent.log${NC}"
echo
echo -e "${YELLOW}Directory Structure:${NC}"
echo -e "  Config:  $CONFIG_DIR"
echo -e "  Data:    $DATA_DIR"
echo -e "  Logs:    $LOG_DIR"
echo -e "  Binary:  $INSTALL_DIR/watchflare-agent"
echo -e "  Service: $PLIST_FILE"
echo
echo -e "${YELLOW}Service Management:${NC}"
echo -e "  Load (start):    ${BLUE}sudo launchctl load $PLIST_FILE${NC}"
echo -e "  Unload (stop):   ${BLUE}sudo launchctl unload $PLIST_FILE${NC}"
echo -e "  Check status:    ${BLUE}sudo launchctl list | grep watchflare${NC}"
echo
