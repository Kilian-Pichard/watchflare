// +build linux

package install

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	systemdServiceFile = "/etc/systemd/system/watchflare-agent.service"
	serviceName        = "watchflare-agent"
)

// LinuxService implements ServiceManager for Linux (systemd)
type LinuxService struct{}

// NewLinuxService creates a new Linux service manager
func NewLinuxService() *LinuxService {
	return &LinuxService{}
}

// Install installs the systemd service
func (s *LinuxService) Install() error {
	// Check if systemd is available
	if !s.hasSystemd() {
		return fmt.Errorf("systemd not available (container environment?)")
	}

	// Create service file content
	serviceContent := `[Unit]
Description=Watchflare Monitoring Agent
Documentation=https://watchflare.io
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=watchflare
Group=watchflare

# Binary
ExecStart=/usr/local/bin/watchflare-agent

# Restart policy
Restart=always
RestartSec=5s

# Environment variables
Environment="WATCHFLARE_CONFIG_DIR=/etc/watchflare"
Environment="WATCHFLARE_DATA_DIR=/var/lib/watchflare"

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/watchflare /var/log

# Logging
StandardOutput=append:/var/log/watchflare-agent.log
StandardError=append:/var/log/watchflare-agent.log
SyslogIdentifier=watchflare-agent

# Resource limits (optional)
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`

	// Write service file
	if err := os.WriteFile(systemdServiceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Set ownership
	if err := os.Chown(systemdServiceFile, 0, 0); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	fmt.Printf("  → Installed to %s\n", systemdServiceFile)

	// Reload systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Println("  → Systemd daemon reloaded")
	return nil
}

// Uninstall removes the systemd service
func (s *LinuxService) Uninstall() error {
	if !s.hasSystemd() {
		return nil // Nothing to do
	}

	// Stop service if running
	if s.IsRunning() {
		if err := s.Stop(); err != nil {
			return err
		}
	}

	// Disable service
	exec.Command("systemctl", "disable", serviceName).Run()

	// Remove service file
	if err := os.Remove(systemdServiceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Println("  → Removed systemd service")
	return nil
}

// Start starts the service
func (s *LinuxService) Start() error {
	if !s.hasSystemd() {
		return fmt.Errorf("systemd not available")
	}

	cmd := exec.Command("systemctl", "start", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Println("  → Service started")
	return nil
}

// Stop stops the service
func (s *LinuxService) Stop() error {
	if !s.hasSystemd() {
		return fmt.Errorf("systemd not available")
	}

	cmd := exec.Command("systemctl", "stop", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Println("  → Service stopped")
	return nil
}

// Enable enables the service to start on boot
func (s *LinuxService) Enable() error {
	if !s.hasSystemd() {
		return fmt.Errorf("systemd not available")
	}

	cmd := exec.Command("systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Println("  → Service enabled (will start on boot)")
	return nil
}

// IsInstalled checks if the service is installed
func (s *LinuxService) IsInstalled() bool {
	_, err := os.Stat(systemdServiceFile)
	return err == nil
}

// IsRunning checks if the service is running
func (s *LinuxService) IsRunning() bool {
	if !s.hasSystemd() {
		return false
	}

	cmd := exec.Command("systemctl", "is-active", "--quiet", serviceName)
	return cmd.Run() == nil
}

// Restart restarts the service
func (s *LinuxService) Restart() error {
	if !s.hasSystemd() {
		return fmt.Errorf("systemd not available")
	}

	fmt.Println("Restarting service...")
	cmd := exec.Command("systemctl", "restart", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	fmt.Println("  → Service restarted")
	return nil
}

// ShowLogs displays and follows the service logs
func (s *LinuxService) ShowLogs() error {
	if !s.hasSystemd() {
		return fmt.Errorf("systemd not available")
	}

	fmt.Println("Following logs (Ctrl+C to exit)...\n")

	cmd := exec.Command("journalctl", "-u", serviceName, "-f", "--no-pager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// hasSystemd checks if systemd is available
func (s *LinuxService) hasSystemd() bool {
	// Check if systemctl command exists
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}

	// Check if systemd is running
	cmd := exec.Command("systemctl", "is-system-running")
	return cmd.Run() == nil || cmd.ProcessState != nil
}
