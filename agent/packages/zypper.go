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

const zypperTimeout = 60 * time.Second

// ZypperCollector collects packages from zypper (openSUSE)
type ZypperCollector struct {
	zypperPath string
}

// Name returns the collector name
func (z *ZypperCollector) Name() string {
	return "zypper"
}

// IsAvailable checks if zypper is available
func (z *ZypperCollector) IsAvailable() bool {
	path, err := exec.LookPath("zypper")
	if err != nil {
		return false
	}
	z.zypperPath = path
	return true
}

// Collect gathers all installed packages from zypper.
// Uses "zypper --no-refresh search --installed-only --details".
// Output format: S | Name | Type | Version | Arch | Repository
func (z *ZypperCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), zypperTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, z.zypperPath, "--no-refresh", "search", "--installed-only", "--details")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("zypper search failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	headerPassed := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !headerPassed {
			if strings.Contains(line, "---+---") {
				headerPassed = true
			}
			continue
		}
		if pkg := parseZypperLine(line); pkg != nil {
			packages = append(packages, pkg)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading zypper output: %w", err)
	}

	return packages, nil
}

// parseZypperLine parses a single data line of "zypper search --details" output.
// Format: "i | name | package | version | arch | repo"
// Only "package" type entries with status "i" (installed) are returned.
func parseZypperLine(line string) *Package {
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

	if status != "i" || pkgType != "package" {
		return nil
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   arch,
		PackageManager: "zypper",
		Source:         repo,
	}
}
