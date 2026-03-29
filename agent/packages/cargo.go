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

const cargoTimeout = 30 * time.Second

// CargoCollector collects installed Rust cargo packages (cross-platform)
type CargoCollector struct {
	cargoPath string
}

// Name returns the collector name
func (c *CargoCollector) Name() string {
	return "cargo"
}

// IsAvailable checks if cargo is available
func (c *CargoCollector) IsAvailable() bool {
	cargoPath, err := exec.LookPath("cargo")
	if err != nil {
		return false
	}
	c.cargoPath = cargoPath
	return true
}

// Collect gathers all installed cargo packages.
// Parses "cargo install --list" output:
//
//	ripgrep v13.0.0:
//	    ripgrep
//	fd-find v9.0.0:
//	    fd
func (c *CargoCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cargoTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.cargoPath, "install", "--list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cargo install --list failed: %w (output: %s)", err, string(output))
	}

	return parseCargoOutput(output), nil
}

// parseCargoOutput parses the output of "cargo install --list".
// Package lines have no leading whitespace: "<name> v<version>:"
// Binary lines are indented and are skipped.
func parseCargoOutput(output []byte) []*Package {
	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		// Package lines start without whitespace: "<name> v<version>:"
		// Binary lines start with whitespace — skip them.
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		version := strings.TrimPrefix(strings.TrimSuffix(parts[1], ":"), "v")

		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			PackageManager: "cargo",
			Source:         "crates.io",
		})
	}

	return packages
}
