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

const nixTimeout = 30 * time.Second

// NixCollector collects packages from Nix package manager (NixOS and multi-distro)
type NixCollector struct {
	nixEnvPath string
}

// Name returns the collector name
func (n *NixCollector) Name() string {
	return "nix"
}

// IsAvailable checks if nix-env is available and accessible.
// nix-env requires access to the Nix daemon socket — service users without
// the required permissions will get "Permission denied" even if the binary exists.
func (n *NixCollector) IsAvailable() bool {
	nixEnvPath, err := exec.LookPath("nix-env")
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, nixEnvPath, "--query", "--installed").Run(); err != nil {
		return false
	}
	n.nixEnvPath = nixEnvPath
	return true
}

// Collect gathers all installed Nix packages.
// Parses "nix-env --query --installed --out-path" output:
//
//	firefox-121.0  /nix/store/abc123-firefox-121.0
func (n *NixCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), nixTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, n.nixEnvPath, "--query", "--installed", "--out-path")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nix-env query failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		if pkg := parseNixLine(scanner.Text()); pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read nix-env output: %w", err)
	}

	return packages, nil
}

// parseNixLine parses a single line of nix-env --query --installed --out-path output.
// Format: "name-version  /nix/store/hash-name-version"
func parseNixLine(line string) *Package {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}

	name, version := splitNixNameVersion(fields[0])
	if name == "" {
		return nil
	}

	storePath := ""
	if len(fields) >= 2 {
		storePath = fields[1]
	}

	return &Package{
		Name:           name,
		Version:        version,
		PackageManager: "nix",
		Source:         "nix-store",
		Description:    storePath,
	}
}

// splitNixNameVersion splits a Nix package string "name-version" into its parts.
// Version is identified as the suffix starting from the last component that
// begins with a digit.
// e.g., "firefox-121.0" → ("firefox", "121.0")
// e.g., "lib32-glibc-2.38" → ("lib32-glibc", "2.38")
func splitNixNameVersion(fullName string) (string, string) {
	parts := strings.Split(fullName, "-")
	if len(parts) < 2 {
		return fullName, ""
	}

	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) > 0 && parts[i][0] >= '0' && parts[i][0] <= '9' {
			return strings.Join(parts[:i], "-"), strings.Join(parts[i:], "-")
		}
	}

	return fullName, ""
}
