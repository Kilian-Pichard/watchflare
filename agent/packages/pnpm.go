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

const pnpmTimeout = 30 * time.Second

// PnpmCollector collects globally installed pnpm packages
type PnpmCollector struct {
	pnpmPath string
}

// Name returns the collector name
func (p *PnpmCollector) Name() string {
	return "pnpm-global"
}

// IsAvailable checks if pnpm is available
func (p *PnpmCollector) IsAvailable() bool {
	path, err := exec.LookPath("pnpm")
	if err != nil {
		return false
	}
	p.pnpmPath = path
	return true
}

// Collect gathers all globally installed pnpm packages.
// Uses "pnpm list -g --depth 0".
func (p *PnpmCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pnpmTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.pnpmPath, "list", "-g", "--depth", "0")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pnpm list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if pkg := parsePnpmLine(scanner.Text()); pkg != nil {
			packages = append(packages, pkg)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading pnpm output: %w", err)
	}

	return packages, nil
}

// parsePnpmLine parses a single line of "pnpm list -g --depth 0" output.
// Handles tree-decorated lines (├──, └──) and formats: "name version", "@scope/name version", "name@version".
func parsePnpmLine(line string) *Package {
	// Remove tree decoration characters
	line = strings.TrimLeft(line, " ├─└│")
	line = strings.TrimSpace(line)

	if line == "" {
		return nil
	}

	// Skip header/section lines (e.g. "dependencies:", "Legend:", paths)
	if strings.HasSuffix(line, ":") || strings.HasPrefix(line, "/") {
		return nil
	}
	if strings.HasPrefix(line, "Legend:") {
		return nil
	}

	var name, version string

	if strings.HasPrefix(line, "@") {
		// Scoped package: @scope/name version
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil
		}
		name = fields[0]
		version = fields[1]
	} else if idx := strings.Index(line, "@"); idx > 0 {
		// name@version format
		name = line[:idx]
		rest := strings.Fields(line[idx+1:])
		if len(rest) > 0 {
			version = rest[0]
		}
	} else {
		// name version format
		fields := strings.Fields(line)
		if len(fields) < 1 {
			return nil
		}
		name = fields[0]
		if len(fields) >= 2 {
			version = fields[1]
		}
	}

	if name == "" {
		return nil
	}

	return &Package{
		Name:           name,
		Version:        version,
		PackageManager: "pnpm-global",
		Source:         "npmjs.com",
	}
}
