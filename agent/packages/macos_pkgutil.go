package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MacOSPkgutilCollector collects system packages installed via .pkg
type MacOSPkgutilCollector struct{}

// Name returns the collector name
func (m *MacOSPkgutilCollector) Name() string {
	return "macos-pkgutil"
}

// IsAvailable checks if pkgutil is available (always on macOS)
func (m *MacOSPkgutilCollector) IsAvailable() bool {
	_, err := exec.LookPath("pkgutil")
	return err == nil
}

// Collect gathers all installed packages via pkgutil
func (m *MacOSPkgutilCollector) Collect() ([]*Package, error) {
	// Get list of all installed packages
	cmd := exec.Command("pkgutil", "--pkgs")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pkgutil --pkgs failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		packageID := strings.TrimSpace(scanner.Text())
		if packageID == "" {
			continue
		}

		// Get package info
		pkg, err := m.getPackageInfo(packageID)
		if err != nil {
			// Skip packages that fail to query
			continue
		}

		packages = append(packages, pkg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse pkgutil output: %w", err)
	}

	return packages, nil
}

// getPackageInfo retrieves detailed information for a package
func (m *MacOSPkgutilCollector) getPackageInfo(packageID string) (*Package, error) {
	// Get package info
	cmd := exec.Command("pkgutil", "--pkg-info", packageID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pkgutil --pkg-info failed for %s: %w", packageID, err)
	}

	// Parse output
	version := ""
	var installedAt time.Time
	var packageSize int64

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "version":
			version = value
		case "install-time":
			// Parse Unix timestamp
			var timestamp int64
			fmt.Sscanf(value, "%d", &timestamp)
			installedAt = time.Unix(timestamp, 0)
		case "volume":
			// Could extract volume info
		case "location":
			// Could extract install location
		}
	}

	if version == "" {
		version = "unknown"
	}

	// Determine source based on package ID prefix
	source := m.detectSource(packageID)

	// Get package size using pkgutil --files
	packageSize = m.calculatePackageSize(packageID)

	// Extract readable name from package ID
	// e.g., "com.apple.pkg.Safari" -> "Safari"
	name := m.extractPackageName(packageID)

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",              // pkgutil doesn't provide this
		PackageManager: "macos-pkgutil",
		Source:         source,
		InstalledAt:    installedAt,
		PackageSize:    packageSize,
		Description:    packageID, // Store full package ID as description
	}, nil
}

// detectSource determines package origin based on package ID
func (m *MacOSPkgutilCollector) detectSource(packageID string) string {
	// Apple packages
	if strings.HasPrefix(packageID, "com.apple.") {
		return "apple"
	}

	// Oracle packages (Java, VirtualBox, etc.)
	if strings.HasPrefix(packageID, "com.oracle.") {
		return "oracle"
	}

	// Adobe packages
	if strings.HasPrefix(packageID, "com.adobe.") {
		return "adobe"
	}

	// Microsoft packages
	if strings.HasPrefix(packageID, "com.microsoft.") {
		return "microsoft"
	}

	// Google packages
	if strings.HasPrefix(packageID, "com.google.") {
		return "google"
	}

	// Docker
	if strings.HasPrefix(packageID, "com.docker.") {
		return "docker"
	}

	return "third-party"
}

// extractPackageName extracts a readable name from package ID
func (m *MacOSPkgutilCollector) extractPackageName(packageID string) string {
	// Try to extract last component of reverse domain name
	// e.g., "com.apple.pkg.Safari" -> "Safari"
	parts := strings.Split(packageID, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return packageID
}

// calculatePackageSize estimates package size
func (m *MacOSPkgutilCollector) calculatePackageSize(packageID string) int64 {
	// Get list of files in package
	cmd := exec.Command("pkgutil", "--files", packageID)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Count files as rough estimate (not accurate size)
	fileCount := strings.Count(string(output), "\n")

	// Very rough estimate: 10KB per file average
	// This is not accurate but gives some indication
	return int64(fileCount * 10240)
}
