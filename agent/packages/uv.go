package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// UvCollector collects tools installed via uv (ultra-fast Python package manager)
// uv is a modern Python package and project manager written in Rust
type UvCollector struct{}

// Name returns the collector name
func (u *UvCollector) Name() string {
	return "uv"
}

// IsAvailable checks if uv is available
func (u *UvCollector) IsAvailable() bool {
	_, err := exec.LookPath("uv")
	return err == nil
}

// Collect gathers all installed uv tools
func (u *UvCollector) Collect() ([]*Package, error) {
	// Run uv tool list to get installed tools
	cmd := exec.Command("uv", "tool", "list")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("uv tool list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse uv tool list output
		// Format varies but typically: "tool-name v1.2.3"
		// or "tool-name"
		pkg := u.parseToolLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading uv output: %w", err)
	}

	return packages, nil
}

// parseToolLine parses a single line of uv tool list output
func (u *UvCollector) parseToolLine(line string) *Package {
	// Remove leading/trailing whitespace and dashes
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "No tools") {
		return nil
	}

	// uv tool list format can vary:
	// - "package-name v1.2.3"
	// - "package-name@1.2.3"
	// - Just "package-name"

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}

	name := fields[0]
	version := ""

	// Check for version in different formats
	if len(fields) >= 2 {
		versionField := fields[1]
		// Remove 'v' prefix if present
		if strings.HasPrefix(versionField, "v") {
			version = versionField[1:]
		} else {
			version = versionField
		}
	} else if strings.Contains(name, "@") {
		// Handle package@version format
		parts := strings.Split(name, "@")
		if len(parts) == 2 {
			name = parts[0]
			version = parts[1]
		}
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",
		PackageManager: "uv",
		Source:         "pypi",
		InstalledAt:    time.Time{},
		PackageSize:    0,
		Description:    "Python tool installed via uv",
	}
}
