package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// FlatpakCollector collects Flatpak packages (cross-distribution)
type FlatpakCollector struct{}

// Name returns the collector name
func (f *FlatpakCollector) Name() string {
	return "flatpak"
}

// IsAvailable checks if flatpak is available
func (f *FlatpakCollector) IsAvailable() bool {
	_, err := exec.LookPath("flatpak")
	return err == nil
}

// Collect gathers all installed flatpak packages
func (f *FlatpakCollector) Collect() ([]*Package, error) {
	// Run flatpak list with detailed output
	// Format: Name\tApplication ID\tVersion\tBranch\tInstallation\tArch
	cmd := exec.Command("flatpak", "list", "--app", "--columns=name,application,version,branch,origin,arch")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("flatpak list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse tab-separated output
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}

		name := strings.TrimSpace(fields[0])
		appID := ""
		version := ""
		branch := ""
		origin := ""
		arch := ""

		if len(fields) >= 2 {
			appID = strings.TrimSpace(fields[1])
		}
		if len(fields) >= 3 {
			version = strings.TrimSpace(fields[2])
		}
		if len(fields) >= 4 {
			branch = strings.TrimSpace(fields[3])
		}
		if len(fields) >= 5 {
			origin = strings.TrimSpace(fields[4])
		}
		if len(fields) >= 6 {
			arch = strings.TrimSpace(fields[5])
		}

		// Use app ID as name if no friendly name
		if name == "" {
			name = appID
		}

		// If no version, use branch
		if version == "" {
			version = branch
		}

		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			Architecture:   arch,
			PackageManager: "flatpak",
			Source:         origin, // Remote repository
			InstalledAt:    time.Time{}, // Not easily available
			PackageSize:    0,          // Would need flatpak info for each package
			Description:    appID,      // Store full app ID as description
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading flatpak output: %w", err)
	}

	return packages, nil
}
