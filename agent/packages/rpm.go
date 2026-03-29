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

const rpmTimeout = 60 * time.Second

// RpmCollector collects packages from rpm (RHEL/CentOS/Fedora)
type RpmCollector struct {
	rpmPath string
}

// Name returns the collector name
func (r *RpmCollector) Name() string {
	return "rpm"
}

// IsAvailable checks if rpm is available
func (r *RpmCollector) IsAvailable() bool {
	path, err := exec.LookPath("rpm")
	if err != nil {
		return false
	}
	r.rpmPath = path
	return true
}

// Collect gathers all installed packages from rpm.
// Uses "rpm -qa" with a custom query format: Name|Version-Release|Arch|Size|InstallTime|Summary.
func (r *RpmCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpmTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.rpmPath, "-qa",
		"--queryformat", "%{NAME}|%{VERSION}-%{RELEASE}|%{ARCH}|%{SIZE}|%{INSTALLTIME}|%{SUMMARY}\n")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rpm query failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if pkg := parseRpmLine(scanner.Text()); pkg != nil {
			packages = append(packages, pkg)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse rpm output: %w", err)
	}

	return packages, nil
}

// parseRpmLine parses a single rpm query output line.
// Format: "name|version-release|arch|size|installtime|summary"
func parseRpmLine(line string) *Package {
	if line == "" {
		return nil
	}
	fields := strings.Split(line, "|")
	if len(fields) < 6 {
		return nil
	}

	var installedAt time.Time
	if ts := parseInt64(fields[4]); ts > 0 {
		installedAt = time.Unix(ts, 0)
	}

	return &Package{
		Name:           fields[0],
		Version:        fields[1],
		Architecture:   fields[2],
		PackageManager: "rpm",
		InstalledAt:    installedAt,
		PackageSize:    parseInt64(fields[3]),
		Description:    TruncateDescription(fields[5]),
	}
}
