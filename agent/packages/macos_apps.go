package packages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"howett.net/plist"
)

// MacOSAppsCollector collects all installed macOS applications
type MacOSAppsCollector struct{}

// Name returns the collector name
func (m *MacOSAppsCollector) Name() string {
	return "macos-apps"
}

// IsAvailable checks if this is macOS
func (m *MacOSAppsCollector) IsAvailable() bool {
	return true // Always available on macOS (registered only for darwin)
}

// appInfo represents app metadata from Info.plist
type appInfo struct {
	CFBundleName            string `plist:"CFBundleName"`
	CFBundleDisplayName     string `plist:"CFBundleDisplayName"`
	CFBundleShortVersionString string `plist:"CFBundleShortVersionString"`
	CFBundleVersion         string `plist:"CFBundleVersion"`
	CFBundleIdentifier      string `plist:"CFBundleIdentifier"`
}

// Collect gathers all installed macOS applications
func (m *MacOSAppsCollector) Collect() ([]*Package, error) {
	var allApps []*Package

	// Collect from /Applications
	systemApps, err := m.collectFromDirectory("/Applications")
	if err != nil {
		return nil, fmt.Errorf("failed to collect system apps: %w", err)
	}
	allApps = append(allApps, systemApps...)

	// Collect from ~/Applications (user apps)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userApps, err := m.collectFromDirectory(filepath.Join(homeDir, "Applications"))
		if err == nil {
			allApps = append(allApps, userApps...)
		}
	}

	return allApps, nil
}

// collectFromDirectory scans a directory for .app bundles
func (m *MacOSAppsCollector) collectFromDirectory(dir string) ([]*Package, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []*Package{}, nil
	}

	var apps []*Package

	// List all .app bundles in directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".app") {
			continue
		}

		appPath := filepath.Join(dir, entry.Name())
		pkg, err := m.getAppInfo(appPath)
		if err != nil {
			// Skip apps that fail to parse
			continue
		}

		apps = append(apps, pkg)
	}

	return apps, nil
}

// getAppInfo extracts metadata from an .app bundle
func (m *MacOSAppsCollector) getAppInfo(appPath string) (*Package, error) {
	// Read Info.plist
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	plistData, err := os.ReadFile(plistPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Info.plist: %w", err)
	}

	var info appInfo
	decoder := plist.NewDecoder(strings.NewReader(string(plistData)))
	if err := decoder.Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse plist: %w", err)
	}

	// Determine app name (prefer display name, fallback to bundle name)
	appName := info.CFBundleDisplayName
	if appName == "" {
		appName = info.CFBundleName
	}
	if appName == "" {
		// Use filename without .app
		appName = strings.TrimSuffix(filepath.Base(appPath), ".app")
	}

	// Determine version
	version := info.CFBundleShortVersionString
	if version == "" {
		version = info.CFBundleVersion
	}
	if version == "" {
		version = "unknown"
	}

	// Detect source (App Store vs Manual)
	source := m.detectSource(appPath)

	// Get install date (modification time of .app bundle)
	var installedAt time.Time
	if stat, err := os.Stat(appPath); err == nil {
		installedAt = stat.ModTime()
	}

	// Calculate approximate size
	size := m.calculateSize(appPath)

	return &Package{
		Name:           appName,
		Version:        version,
		Architecture:   "",           // Could detect Universal/ARM/Intel later
		PackageManager: "macos-apps",
		Source:         source,
		InstalledAt:    installedAt,
		PackageSize:    size,
		Description:    info.CFBundleIdentifier, // Store bundle ID as description for now
	}, nil
}

// detectSource determines if app is from App Store or installed manually
func (m *MacOSAppsCollector) detectSource(appPath string) string {
	// Check for _MASReceipt (Mac App Store receipt)
	receiptPath := filepath.Join(appPath, "Contents", "_MASReceipt", "receipt")
	if _, err := os.Stat(receiptPath); err == nil {
		return "app-store"
	}

	// Check if it's in /Applications/Utilities (system utilities)
	if strings.Contains(appPath, "/Applications/Utilities") {
		return "system"
	}

	// Check if it's a known system app
	appName := filepath.Base(appPath)
	systemApps := []string{
		"Safari.app", "Mail.app", "Calendar.app", "Contacts.app",
		"Notes.app", "Photos.app", "Music.app", "TV.app",
		"Podcasts.app", "Messages.app", "FaceTime.app",
		"Maps.app", "Books.app", "News.app", "Stocks.app",
		"Home.app", "Voice Memos.app", "Reminders.app",
		"Clock.app", "Weather.app", "Translate.app",
		"App Store.app", "System Settings.app", "Time Machine.app",
		"QuickTime Player.app", "Preview.app", "TextEdit.app",
		"Font Book.app", "Image Capture.app", "Dictionary.app",
	}

	for _, sysApp := range systemApps {
		if appName == sysApp {
			return "system"
		}
	}

	return "manual"
}

// calculateSize estimates the size of an app bundle
func (m *MacOSAppsCollector) calculateSize(appPath string) int64 {
	var totalSize int64

	// Use du command for accurate size
	cmd := exec.Command("du", "-sk", appPath)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse output (format: "12345\t/path/to/app")
	parts := strings.Fields(string(output))
	if len(parts) > 0 {
		var sizeKB int64
		fmt.Sscanf(parts[0], "%d", &sizeKB)
		totalSize = sizeKB * 1024 // Convert to bytes
	}

	return totalSize
}
