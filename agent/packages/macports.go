package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"
)

const macportsTimeout = 30 * time.Second

// portLineRegex matches installed port lines: "  name @version[+variants] [(active)]"
var portLineRegex = regexp.MustCompile(`^\s+(\S+)\s+@([^\s+]+)`)

// MacPortsCollector collects packages from MacPorts (macOS)
type MacPortsCollector struct {
	portPath string
}

// Name returns the collector name
func (m *MacPortsCollector) Name() string {
	return "macports"
}

// IsAvailable checks if port is available
func (m *MacPortsCollector) IsAvailable() bool {
	portPath := "/opt/local/bin/port"
	if _, err := os.Stat(portPath); err == nil {
		m.portPath = portPath
		return true
	}
	return false
}

// Collect gathers all installed packages from MacPorts.
// Parses "port installed" output:
//
//	git @2.43.0_0+credential_osxkeychain (active)
func (m *MacPortsCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), macportsTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.portPath, "installed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("port installed failed: %w (output: %s)", err, string(output))
	}

	var pkgs []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		if pkg := parsePortLine(scanner.Text()); pkg != nil {
			pkgs = append(pkgs, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse port output: %w", err)
	}

	return pkgs, nil
}

// parsePortLine parses a single line of port installed output.
// Package lines are indented; header and blank lines are skipped.
func parsePortLine(line string) *Package {
	matches := portLineRegex.FindStringSubmatch(line)
	if len(matches) < 3 {
		return nil
	}

	return &Package{
		Name:           matches[1],
		Version:        matches[2],
		PackageManager: "macports",
		Source:         "macports.org",
	}
}
