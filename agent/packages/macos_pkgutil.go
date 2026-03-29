package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	pkgutilListTimeout = 30 * time.Second
	pkgutilInfoTimeout = 5 * time.Second
)

// MacOSPkgutilCollector collects system packages installed via .pkg
type MacOSPkgutilCollector struct {
	pkgutilPath string
}

// Name returns the collector name
func (m *MacOSPkgutilCollector) Name() string {
	return "macos-pkgutil"
}

// IsAvailable checks if pkgutil is available (always on macOS)
func (m *MacOSPkgutilCollector) IsAvailable() bool {
	pkgutilPath, err := exec.LookPath("pkgutil")
	if err != nil {
		return false
	}
	m.pkgutilPath = pkgutilPath
	return true
}

// Collect gathers all installed packages via pkgutil
func (m *MacOSPkgutilCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pkgutilListTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.pkgutilPath, "--pkgs")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pkgutil --pkgs failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		packageID := strings.TrimSpace(scanner.Text())
		if packageID == "" {
			continue
		}
		pkg, err := m.getPackageInfo(packageID)
		if err != nil {
			continue
		}
		packages = append(packages, pkg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse pkgutil output: %w", err)
	}

	return packages, nil
}

// getPackageInfo retrieves detailed information for a single package
func (m *MacOSPkgutilCollector) getPackageInfo(packageID string) (*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pkgutilInfoTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.pkgutilPath, "--pkg-info", packageID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pkgutil --pkg-info failed for %s: %w", packageID, err)
	}

	version, installedAt := parsePkgInfo(output)
	if version == "" {
		version = "unknown"
	}

	return &Package{
		Name:           extractPkgName(packageID),
		Version:        version,
		PackageManager: "macos-pkgutil",
		Source:         detectPkgSource(packageID),
		InstalledAt:    installedAt,
		Description:    packageID,
	}, nil
}

// parsePkgInfo extracts version and install time from pkgutil --pkg-info output.
func parsePkgInfo(output []byte) (version string, installedAt time.Time) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "version":
			version = value
		case "install-time":
			var timestamp int64
			fmt.Sscanf(value, "%d", &timestamp)
			if timestamp > 0 {
				installedAt = time.Unix(timestamp, 0)
			}
		}
	}
	return
}

// extractPkgName extracts a readable name from a reverse-domain package ID.
// e.g., "com.apple.pkg.Safari" → "Safari"
func extractPkgName(packageID string) string {
	parts := strings.Split(packageID, ".")
	return parts[len(parts)-1]
}

// detectPkgSource determines package origin based on package ID prefix.
func detectPkgSource(packageID string) string {
	switch {
	case strings.HasPrefix(packageID, "com.apple."):
		return "apple"
	case strings.HasPrefix(packageID, "com.oracle."):
		return "oracle"
	case strings.HasPrefix(packageID, "com.adobe."):
		return "adobe"
	case strings.HasPrefix(packageID, "com.microsoft."):
		return "microsoft"
	case strings.HasPrefix(packageID, "com.google."):
		return "google"
	case strings.HasPrefix(packageID, "com.docker."):
		return "docker"
	default:
		return "third-party"
	}
}
