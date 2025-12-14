# Dashboard - Installation Instructions UI

## 📊 Interface après création du serveur

Quand l'utilisateur crée un serveur, il arrive sur cette page:

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  ✓ Server Created Successfully!                                    │
│                                                                     │
│  Server "web-server-01" has been created with status: pending      │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  📦 Installation Instructions                                       │
│                                                                     │
│  Choose your operating system:                                     │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │  🐧      │  │  🍎      │  │  🪟      │  │  🐳      │          │
│  │  Linux   │  │  macOS   │  │ Windows  │  │  Docker  │          │
│  │  (Active)│  │          │  │ (Soon)   │  │ (Soon)   │          │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘          │
│                                                                     │
│  ⚠️ Important: Save your registration token securely!              │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ Registration Token:                                         │  │
│  │ ┌───────────────────────────────────────────────┬────────┐  │  │
│  │ │ wf_reg_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7...  │ [Copy] │  │  │
│  │ └───────────────────────────────────────────────┴────────┘  │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                                     │
│  🔹 Quick Install (Recommended)                                    │
│                                                                     │
│  Run this command on your server:                                  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ $ curl -sSL https://get.watchflare.io/ | sudo bash -s -- \ │  │
│  │     --token wf_reg_a1b2c3d4e5f6... \                       │  │
│  │     --host backend.watchflare.com                           │  │
│  │                                               [Copy Command]│  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ✓ Automatically detects your operating system                     │
│  ✓ Downloads and installs Watchflare Agent                         │
│  ✓ Registers your server automatically                             │
│  ✓ Starts the monitoring service                                   │
│                                                                     │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                                     │
│  🔹 Platform-Specific Installation (Advanced)                      │
│                                                                     │
│  [ Show platform-specific commands ▼ ]                             │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ 🐧 Linux:                                                   │  │
│  │ curl -sSL https://get.watchflare.io/linux | sudo bash -s \ │  │
│  │   --token wf_reg_a1b2c3d4e5f6... --host backend...         │  │
│  │                                               [Copy Command]│  │
│  │                                                             │  │
│  │ 🍎 macOS:                                                   │  │
│  │ curl -sSL https://get.watchflare.io/macos | sudo bash -s \ │  │
│  │   --token wf_reg_a1b2c3d4e5f6... --host backend...         │  │
│  │                                               [Copy Command]│  │
│  │                                                             │  │
│  │ 🪟 Windows (Coming Soon):                                   │  │
│  │ irm get.watchflare.io/windows | iex                        │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                                     │
│  🔹 Manual Installation                                            │
│                                                                     │
│  [ Show manual installation steps ▼ ]                              │
│                                                                     │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                                     │
│  📖 What happens next?                                              │
│                                                                     │
│  1. The agent will register with this server                       │
│  2. Status will change from "pending" to "online"                  │
│  3. You'll start receiving metrics and heartbeats                  │
│  4. This page will update automatically when connected             │
│                                                                     │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                                     │
│  [ View Server Details ]  [ Back to Servers List ]                 │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## 🐧 Linux Tab (Active par défaut)

```
╔═════════════════════════════════════════════════════════════════╗
║  🐧 Linux Installation                                          ║
╚═════════════════════════════════════════════════════════════════╝

🚀 Quick Install (Recommended)
─────────────────────────────────────────────────────────────────

curl -sSL https://get.watchflare.io/ | sudo bash -s -- \
  --token wf_reg_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7... \
  --host backend.watchflare.com

                                                    [Copy Command]

Supported distributions:
✓ Ubuntu 18.04+     ✓ Debian 10+       ✓ CentOS 7+
✓ RHEL 7+           ✓ Fedora 30+       ✓ Amazon Linux 2


📦 Manual Installation
─────────────────────────────────────────────────────────────────

1. Download the agent:
   wget https://github.com/watchflare/watchflare/releases/latest/download/watchflare-agent-linux-amd64
   chmod +x watchflare-agent-linux-amd64

2. Run the installer:
   sudo ./install.sh

3. Register the agent:
   sudo watchflare-agent \
     --token wf_reg_a1b2c3d4e5f6... \
     --host backend.watchflare.com

4. Start the service:
   sudo systemctl start watchflare-agent
   sudo systemctl enable watchflare-agent


🔍 Verify Installation
─────────────────────────────────────────────────────────────────

sudo systemctl status watchflare-agent
sudo journalctl -u watchflare-agent -f


📁 File Locations
─────────────────────────────────────────────────────────────────

Config:  /etc/watchflare/agent.conf
Logs:    /var/log/watchflare/
Data:    /var/lib/watchflare/
Service: /etc/systemd/system/watchflare-agent.service
```

## 🍎 macOS Tab

```
╔═════════════════════════════════════════════════════════════════╗
║  🍎 macOS Installation                                          ║
╚═════════════════════════════════════════════════════════════════╝

🚀 Quick Install (Recommended)
─────────────────────────────────────────────────────────────────

curl -sSL https://get.watchflare.io/ | sudo bash -s -- \
  --token wf_reg_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7... \
  --host backend.watchflare.com

                                                    [Copy Command]

Supported versions:
✓ macOS 11 (Big Sur) and later
✓ Intel and Apple Silicon (M1/M2/M3)


📦 Manual Installation
─────────────────────────────────────────────────────────────────

1. Download the agent:
   # For Apple Silicon (M1/M2/M3):
   curl -LO https://github.com/watchflare/watchflare/releases/latest/download/watchflare-agent-darwin-arm64

   # For Intel Macs:
   curl -LO https://github.com/watchflare/watchflare/releases/latest/download/watchflare-agent-darwin-amd64

2. Make executable and install:
   chmod +x watchflare-agent-*
   sudo ./install-macos.sh

3. Register the agent:
   sudo watchflare-agent \
     --token wf_reg_a1b2c3d4e5f6... \
     --host backend.watchflare.com

4. Start the service:
   sudo launchctl load /Library/LaunchDaemons/com.watchflare.agent.plist


🔍 Verify Installation
─────────────────────────────────────────────────────────────────

sudo launchctl list | grep watchflare
tail -f /var/log/watchflare/watchflare-agent.log


📁 File Locations
─────────────────────────────────────────────────────────────────

Config:  /etc/watchflare/agent.conf
Logs:    /var/log/watchflare/
Data:    /var/lib/watchflare/
Service: /Library/LaunchDaemons/com.watchflare.agent.plist
```

## 🪟 Windows Tab (Coming Soon)

```
╔═════════════════════════════════════════════════════════════════╗
║  🪟 Windows Installation (Coming Soon)                          ║
╚═════════════════════════════════════════════════════════════════╝

Windows support is coming soon!

In the meantime, you can:
• Use Windows Subsystem for Linux (WSL) with the Linux installer
• Run the Docker container on Windows
• Stay tuned for native Windows agent release

[ Notify me when available ]
```

## 🐳 Docker Tab (Coming Soon)

```
╔═════════════════════════════════════════════════════════════════╗
║  🐳 Docker Installation (Coming Soon)                           ║
╚═════════════════════════════════════════════════════════════════╝

Docker support is coming soon!

We're working on:
• Docker container image
• Docker Compose example
• Kubernetes deployment

[ Notify me when available ]
```

## 🎨 Design Notes for Frontend Team

### Color Scheme
- Success: Green (#2f855a)
- Warning: Yellow/Orange (#f6ad55)
- Info: Blue (#667eea)
- Code blocks: Dark (#1a202c)

### Interactive Elements
1. **Tab Navigation**
   - Active tab: Blue underline + darker background
   - Hover: Subtle background change
   - Disabled tabs (Windows, Docker): Gray with "Soon" badge

2. **Copy Buttons**
   - Icon: 📋 or copy icon
   - Click feedback: "Copied!" message (2s)
   - Hover: Tooltip "Copy to clipboard"

3. **Collapsible Sections**
   - "Show manual installation" accordion
   - Smooth expand/collapse animation
   - Chevron icon rotates

4. **Real-time Status**
   - Poll /servers/:id every 5s
   - When status changes to "online":
     - Show success banner
     - Confetti animation
     - "Agent Connected!" message

### Responsive Design
- Mobile: Stack tabs vertically
- Tablet: 2 columns for tabs
- Desktop: 4 columns for tabs

### Code Blocks
- Syntax highlighting
- Line numbers optional
- Scroll for long commands
- Copy button in top-right corner
