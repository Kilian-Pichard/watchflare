package packages

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

// Collect gathers all installed cargo packages
func (c *CargoCollector) Collect() ([]*Package, error) {
	// Run cargo install --list to get installed packages
	cmd := exec.Command(c.cargoPath, "install", "--list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cargo install --list failed: %w (output: %s)", err, string(output))
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	var currentPackage string
	var currentVersion string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Package lines start without whitespace and contain version
		// Example: "ripgrep v13.0.0:"
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentPackage = parts[0]
				// Version includes 'v' prefix, e.g., "v13.0.0:"
				currentVersion = strings.TrimPrefix(strings.TrimSuffix(parts[1], ":"), "v")

				packages = append(packages, &Package{
					Name:           currentPackage,
					Version:        currentVersion,
					Architecture:   "",           // cargo is platform-independent (compiles locally)
					PackageManager: "cargo",
					Source:         "crates.io",  // Default Rust package registry
					InstalledAt:    time.Time{}, // Would require checking .crates.toml timestamp
					PackageSize:    0,            // Would require scanning ~/.cargo/bin
					Description:    "",           // Would require cargo search call
				})
			}
		}
		// Binary lines start with whitespace (we skip them)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse cargo output: %w", err)
	}

	return packages, nil
}

// getCargoHome returns the cargo home directory
func getCargoHome() string {
	if home := os.Getenv("CARGO_HOME"); home != "" {
		return home
	}

	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".cargo")
	}

	return ""
}
