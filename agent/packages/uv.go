package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const uvTimeout = 30 * time.Second

// UvCollector collects tools installed via uv (Python package manager)
type UvCollector struct {
	uvPath string
}

// Name returns the collector name
func (u *UvCollector) Name() string {
	return "uv"
}

// IsAvailable checks if uv is available
func (u *UvCollector) IsAvailable() bool {
	// Try PATH first, then common user-local install locations
	if path, err := exec.LookPath("uv"); err == nil {
		u.uvPath = path
		return true
	}
	// Fallback paths for common user-local installs when HOME is not in PATH.
	// HOME is checked first so the current user's install always takes precedence.
	seen := make(map[string]bool)
	var userLocalPaths []string
	for _, candidate := range []string{
		os.Getenv("HOME") + "/.local/bin/uv", // current user (macOS + Linux)
		"/home/watchflare/.local/bin/uv",      // watchflare service user (Linux)
		"/root/.local/bin/uv",                 // root user
	} {
		if candidate != "/.local/bin/uv" && !seen[candidate] {
			seen[candidate] = true
			userLocalPaths = append(userLocalPaths, candidate)
		}
	}
	for _, path := range userLocalPaths {
		if _, err := os.Stat(path); err == nil {
			u.uvPath = path
			return true
		}
	}
	return false
}

// Collect gathers all installed uv tools.
// Uses "uv tool list". Format: "name vVersion\n    - cmd\n    - cmd"
func (u *UvCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), uvTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, u.uvPath, "tool", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("uv tool list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if pkg := parseUvLine(scanner.Text()); pkg != nil {
			packages = append(packages, pkg)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading uv output: %w", err)
	}

	return packages, nil
}

// parseUvLine parses a single line of "uv tool list" output.
// Handles formats: "name vVersion", "name@version", "name".
// Skips sub-command lines (indented "- cmd") and "No tools installed" messages.
func parseUvLine(line string) *Package {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "No tools") {
		return nil
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}

	name := fields[0]
	version := ""

	if len(fields) >= 2 {
		v := fields[1]
		version = strings.TrimPrefix(v, "v")
	} else if idx := strings.Index(name, "@"); idx > 0 {
		version = name[idx+1:]
		name = name[:idx]
	}

	return &Package{
		Name:           name,
		Version:        version,
		PackageManager: "uv",
		Source:         "pypi.org",
	}
}
