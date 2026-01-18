package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PnpmCollector collects globally installed pnpm packages
type PnpmCollector struct{}

// Name returns the collector name
func (p *PnpmCollector) Name() string {
	return "pnpm-global"
}

// IsAvailable checks if pnpm is available
func (p *PnpmCollector) IsAvailable() bool {
	_, err := exec.LookPath("pnpm")
	return err == nil
}

// Collect gathers all globally installed pnpm packages
func (p *PnpmCollector) Collect() ([]*Package, error) {
	// Run pnpm list -g --depth 0 to get global packages
	cmd := exec.Command("pnpm", "list", "-g", "--depth", "0")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pnpm list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header lines until we find packages
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse pnpm list output
		// Format varies, but typically:
		// package-name 1.2.3
		// or with tree symbols
		pkg := p.parsePackageLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading pnpm output: %w", err)
	}

	return packages, nil
}

// parsePackageLine parses a single line of pnpm list output
func (p *PnpmCollector) parsePackageLine(line string) *Package {
	// Remove tree characters (├──, └──, │, etc.)
	line = strings.TrimLeft(line, " ├─└│")
	line = strings.TrimSpace(line)

	if line == "" {
		return nil
	}

	// Skip dependency lines (contain "dependencies:" or other headers)
	if strings.Contains(line, ":") && !strings.Contains(line, " ") {
		return nil
	}

	// Parse format: "package-name version" or "package-name@version"
	var name, version string

	if strings.Contains(line, "@") && !strings.HasPrefix(line, "@") {
		// Format: package@version
		parts := strings.Split(line, "@")
		if len(parts) >= 2 {
			name = parts[0]
			version = strings.Fields(parts[1])[0] // Take first field after @
		}
	} else if strings.HasPrefix(line, "@") {
		// Scoped package: @scope/package version
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name = fields[0]
			version = fields[1]
		}
	} else {
		// Format: package version
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name = fields[0]
			version = fields[1]
		} else if len(fields) == 1 {
			name = fields[0]
		}
	}

	if name == "" {
		return nil
	}

	// Filter out non-package lines
	if strings.HasPrefix(name, "Legend:") || strings.HasPrefix(name, "dependencies") {
		return nil
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",
		PackageManager: "pnpm-global",
		Source:         "pnpm",
		InstalledAt:    time.Time{},
		PackageSize:    0,
		Description:    "Globally installed pnpm package",
	}
}
