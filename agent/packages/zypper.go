package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ZypperCollector collects packages from zypper (openSUSE)
type ZypperCollector struct{}

// Name returns the collector name
func (z *ZypperCollector) Name() string {
	return "zypper"
}

// IsAvailable checks if zypper is available
func (z *ZypperCollector) IsAvailable() bool {
	_, err := exec.LookPath("zypper")
	return err == nil
}

// Collect gathers all installed packages from zypper
func (z *ZypperCollector) Collect() ([]*Package, error) {
	// Run zypper search with installed packages
	// --installed-only: only installed packages
	// --details: show detailed information
	cmd := exec.Command("zypper", "--no-refresh", "search", "--installed-only", "--details")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("zypper search failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header lines
	headerPassed := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Skip until we find the separator line (----+----+...)
		if !headerPassed {
			if strings.Contains(line, "---+---") {
				headerPassed = true
			}
			continue
		}

		// Parse package line
		pkg := z.parsePackageLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading zypper output: %w", err)
	}

	return packages, nil
}

// parsePackageLine parses a single line of zypper output
func (z *ZypperCollector) parsePackageLine(line string) *Package {
	// Zypper --details output format (separated by |):
	// S | Name | Type | Version | Arch | Repository
	// i | package-name | package | 1.2.3-1 | x86_64 | repo-name

	parts := strings.Split(line, "|")
	if len(parts) < 6 {
		return nil
	}

	status := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	pkgType := strings.TrimSpace(parts[2])
	version := strings.TrimSpace(parts[3])
	arch := strings.TrimSpace(parts[4])
	repo := strings.TrimSpace(parts[5])

	// Only include installed packages (status 'i')
	if status != "i" {
		return nil
	}

	// Filter to only include actual packages (not patches, patterns, etc.)
	if pkgType != "package" {
		return nil
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   arch,
		PackageManager: "zypper",
		Source:         repo,
		InstalledAt:    time.Time{}, // Not easily available
		PackageSize:    0,            // Would need rpm -qi for each package
		Description:    "",           // Would need rpm -qi for description
	}
}
