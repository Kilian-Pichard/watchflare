package packages

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"howett.net/plist"
)

const macosAppsTimeout = 5 * time.Second

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
	CFBundleName               string `plist:"CFBundleName"`
	CFBundleDisplayName        string `plist:"CFBundleDisplayName"`
	CFBundleShortVersionString string `plist:"CFBundleShortVersionString"`
	CFBundleVersion            string `plist:"CFBundleVersion"`
	CFBundleIdentifier         string `plist:"CFBundleIdentifier"`
}

// Collect gathers all installed macOS applications
func (m *MacOSAppsCollector) Collect() ([]*Package, error) {
	var allApps []*Package

	systemApps, err := m.collectFromDirectory("/Applications")
	if err != nil {
		return nil, fmt.Errorf("failed to collect system apps: %w", err)
	}
	allApps = append(allApps, systemApps...)

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
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []*Package{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var apps []*Package
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".app") {
			continue
		}

		appPath := filepath.Join(dir, entry.Name())
		pkg, err := m.getAppInfo(appPath)
		if err != nil {
			continue
		}
		apps = append(apps, pkg)
	}

	return apps, nil
}

// getAppInfo extracts metadata from an .app bundle
func (m *MacOSAppsCollector) getAppInfo(appPath string) (*Package, error) {
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	plistData, err := os.ReadFile(plistPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Info.plist: %w", err)
	}

	var info appInfo
	if err := plist.NewDecoder(bytes.NewReader(plistData)).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse plist: %w", err)
	}

	appName := appNameFromInfo(info, appPath)
	version := appVersionFromInfo(info)

	var installedAt time.Time
	if stat, err := os.Stat(appPath); err == nil {
		installedAt = stat.ModTime()
	}

	return &Package{
		Name:           appName,
		Version:        version,
		PackageManager: "macos-apps",
		Source:         detectMacOSSource(appPath),
		InstalledAt:    installedAt,
		PackageSize:    calculateMacOSAppSize(appPath),
		Description:    info.CFBundleIdentifier,
	}, nil
}

// appNameFromInfo determines the display name from plist info, falling back to the filename.
func appNameFromInfo(info appInfo, appPath string) string {
	if info.CFBundleDisplayName != "" {
		return info.CFBundleDisplayName
	}
	if info.CFBundleName != "" {
		return info.CFBundleName
	}
	return strings.TrimSuffix(filepath.Base(appPath), ".app")
}

// appVersionFromInfo determines the version from plist info.
func appVersionFromInfo(info appInfo) string {
	if info.CFBundleShortVersionString != "" {
		return info.CFBundleShortVersionString
	}
	if info.CFBundleVersion != "" {
		return info.CFBundleVersion
	}
	return "unknown"
}

// macOSSystemApps is the list of known Apple-bundled applications.
var macOSSystemApps = []string{
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

// detectMacOSSource determines whether an app came from the App Store, is a
// system app, or was installed manually.
func detectMacOSSource(appPath string) string {
	receiptPath := filepath.Join(appPath, "Contents", "_MASReceipt", "receipt")
	if _, err := os.Stat(receiptPath); err == nil {
		return "app-store"
	}
	if strings.Contains(appPath, "/Applications/Utilities") {
		return "system"
	}
	if slices.Contains(macOSSystemApps, filepath.Base(appPath)) {
		return "system"
	}
	return "manual"
}

// calculateMacOSAppSize estimates the size of an app bundle using du.
func calculateMacOSAppSize(appPath string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), macosAppsTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/du", "-sk", appPath)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	parts := strings.Fields(string(output))
	if len(parts) == 0 {
		return 0
	}

	var sizeKB int64
	fmt.Sscanf(parts[0], "%d", &sizeKB)
	return sizeKB * 1024
}
