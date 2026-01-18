package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NixCollector collects packages from Nix package manager (NixOS and multi-distro)
type NixCollector struct{}

// Name returns the collector name
func (n *NixCollector) Name() string {
	return "nix"
}

// IsAvailable checks if nix is available
func (n *NixCollector) IsAvailable() bool {
	_, err := exec.LookPath("nix-env")
	return err == nil
}

// Collect gathers all installed Nix packages
func (n *NixCollector) Collect() ([]*Package, error) {
	// Run nix-env to list installed packages
	// --query: query mode
	// --installed: only installed packages
	// --out-path: show store path
	cmd := exec.Command("nix-env", "--query", "--installed", "--out-path")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nix-env query failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse nix-env output
		// Format: "package-name-version  /nix/store/hash-package-name-version"
		pkg := n.parsePackageLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading nix-env output: %w", err)
	}

	return packages, nil
}

// parsePackageLine parses a single line of nix-env output
func (n *NixCollector) parsePackageLine(line string) *Package {
	// Format: "package-name-version  /nix/store/hash-package-name-version"
	fields := strings.Fields(line)
	if len(fields) < 1 {
		return nil
	}

	fullName := fields[0]
	storePath := ""
	if len(fields) >= 2 {
		storePath = fields[1]
	}

	// Split name and version
	// Nix packages are typically: name-version
	name, version := n.splitNameVersion(fullName)
	if name == "" {
		return nil
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",          // Nix is architecture-independent in naming
		PackageManager: "nix",
		Source:         "nix-store", // Could extract channel info if needed
		InstalledAt:    time.Time{}, // Not easily available
		PackageSize:    0,            // Would need du on store path
		Description:    storePath,    // Store the nix store path
	}
}

// splitNameVersion splits a Nix package string into name and version
func (n *NixCollector) splitNameVersion(fullName string) (string, string) {
	// Nix packages follow pattern: name-version
	// Version typically starts with a digit
	// Example: "firefox-121.0" -> name="firefox", version="121.0"

	parts := strings.Split(fullName, "-")
	if len(parts) < 2 {
		return fullName, ""
	}

	// Find where version starts (first part that starts with a digit)
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) > 0 && parts[i][0] >= '0' && parts[i][0] <= '9' {
			// Found version start
			name := strings.Join(parts[:i], "-")
			version := strings.Join(parts[i:], "-")
			return name, version
		}
	}

	// Couldn't determine version
	return fullName, ""
}
