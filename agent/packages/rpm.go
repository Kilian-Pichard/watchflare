package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RpmCollector collects packages from rpm (RHEL/CentOS/Fedora)
type RpmCollector struct{}

// Name returns the collector name
func (r *RpmCollector) Name() string {
	return "rpm"
}

// IsAvailable checks if rpm is available
func (r *RpmCollector) IsAvailable() bool {
	_, err := exec.LookPath("rpm")
	return err == nil
}

// Collect gathers all installed packages from rpm
func (r *RpmCollector) Collect() ([]*Package, error) {
	// Run rpm query to get package information
	// Format: Name|Version|Architecture|Size|Install Time|Summary
	cmd := exec.Command("rpm", "-qa",
		"--queryformat", "%{NAME}|%{VERSION}-%{RELEASE}|%{ARCH}|%{SIZE}|%{INSTALLTIME}|%{SUMMARY}\n")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rpm query failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 6 {
			continue
		}

		name := fields[0]
		version := fields[1]
		arch := fields[2]
		sizeStr := fields[3]
		installTimeStr := fields[4]
		summary := fields[5]

		// Parse size (rpm reports in bytes)
		size := parseInt64(sizeStr)

		// Parse install time (Unix timestamp)
		var installedAt time.Time
		installTimeUnix := parseInt64(installTimeStr)
		if installTimeUnix > 0 {
			installedAt = time.Unix(installTimeUnix, 0)
		}

		// Truncate summary to 100 chars
		summary = TruncateDescription(summary)

		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			Architecture:   arch,
			PackageManager: "rpm",
			Source:         "", // rpm doesn't easily provide repo info
			InstalledAt:    installedAt,
			PackageSize:    size,
			Description:    summary,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse rpm output: %w", err)
	}

	return packages, nil
}
