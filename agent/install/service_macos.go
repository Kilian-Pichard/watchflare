//go:build darwin

package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	macOSServiceName = "io.watchflare.agent"
	plistPath        = "/Library/LaunchDaemons/io.watchflare.agent.plist"
)

// MacOSService implements ServiceManager for macOS (launchd)
type MacOSService struct{}

// NewMacOSService creates a new macOS service manager
func NewMacOSService() *MacOSService {
	return &MacOSService{}
}

// Install installs the launchd service
func (s *MacOSService) Install() error {
	// Create plist content
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>io.watchflare.agent</string>

	<key>ProgramArguments</key>
	<array>
		<string>/usr/local/bin/watchflare-agent</string>
	</array>

	<key>UserName</key>
	<string>watchflare</string>

	<key>GroupName</key>
	<string>staff</string>

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
		<string>/etc/watchflare</string>
		<key>WATCHFLARE_DATA_DIR</key>
		<string>/var/lib/watchflare</string>
		<key>HOMEBREW_CACHE</key>
		<string>/var/lib/watchflare/brew-cache</string>
	</dict>

	<key>ThrottleInterval</key>
	<integer>5</integer>
</dict>
</plist>
`

	// Write plist file
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Set ownership
	if err := os.Chown(plistPath, 0, 0); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	fmt.Printf("  → Installed to %s\n", plistPath)
	return nil
}

// Uninstall removes the launchd service
func (s *MacOSService) Uninstall() error {
	// Stop service if running
	if s.IsRunning() {
		if err := s.Stop(); err != nil {
			return err
		}
	}

	// Remove plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	fmt.Println("  → Removed launchd service")
	return nil
}

// Start starts the service (bootstrap in launchd terminology)
func (s *MacOSService) Start() error {
	cmd := exec.Command("launchctl", "bootstrap", "system", plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already loaded
		if strings.Contains(string(output), "Already loaded") {
			fmt.Println("  → Service already running")
			return nil
		}
		return fmt.Errorf("failed to start service: %w (output: %s)", err, string(output))
	}

	fmt.Println("  → Service started")
	return nil
}

// Stop stops the service (bootout in launchd terminology)
func (s *MacOSService) Stop() error {
	cmd := exec.Command("launchctl", "bootout", "system/"+macOSServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already stopped
		if strings.Contains(string(output), "Could not find") {
			fmt.Println("  → Service already stopped")
			return nil
		}
		return fmt.Errorf("failed to stop service: %w (output: %s)", err, string(output))
	}

	fmt.Println("  → Service stopped")
	return nil
}

// Enable enables the service (on macOS, services with RunAtLoad=true start automatically)
func (s *MacOSService) Enable() error {
	// On macOS, with RunAtLoad=true, the service is automatically enabled
	// No additional action needed
	fmt.Println("  → Service enabled (RunAtLoad=true)")
	return nil
}

// IsInstalled checks if the service is installed
func (s *MacOSService) IsInstalled() bool {
	_, err := os.Stat(plistPath)
	return err == nil
}

// IsRunning checks if the service is running
func (s *MacOSService) IsRunning() bool {
	cmd := exec.Command("launchctl", "print", "system/"+macOSServiceName)
	return cmd.Run() == nil
}

// Restart restarts the service
func (s *MacOSService) Restart() error {
	fmt.Println("Restarting service...")

	// Use kickstart -k to kill and restart the service (works only if already loaded)
	// This is more reliable than stop + start on macOS
	if s.IsRunning() {
		cmd := exec.Command("launchctl", "kickstart", "-k", "system/"+macOSServiceName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to restart service: %w", err)
		}
		fmt.Println("  → Service restarted")
		return nil
	}

	// If not running, just start it
	return s.Start()
}

// ShowLogs displays and follows the service logs
func (s *MacOSService) ShowLogs() error {
	logPath := "/var/log/watchflare-agent.log"
	fmt.Printf("Following logs from %s (Ctrl+C to exit)...\n\n", logPath)

	cmd := exec.Command("tail", "-f", logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
