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

const flatpakTimeout = 30 * time.Second

// FlatpakCollector collects Flatpak packages (cross-distribution)
type FlatpakCollector struct {
	flatpakPath string
}

// Name returns the collector name
func (f *FlatpakCollector) Name() string {
	return "flatpak"
}

// IsAvailable checks if flatpak is available and accessible.
// flatpak requires a D-Bus session — service users without a session
// will get "Permission denied" even if the binary exists.
func (f *FlatpakCollector) IsAvailable() bool {
	flatpakPath, err := exec.LookPath("flatpak")
	if err != nil {
		return false
	}
	// Probe with a quick non-destructive command to verify D-Bus access.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, flatpakPath, "list", "--app", "--columns=name").Run(); err != nil {
		return false
	}
	f.flatpakPath = flatpakPath
	return true
}

// Collect gathers all installed flatpak applications.
// Parses tab-separated output of:
//
//	flatpak list --app --columns=name,application,version,branch,origin,arch
func (f *FlatpakCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), flatpakTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, f.flatpakPath, "list", "--app",
		"--columns=name,application,version,branch,origin,arch")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("flatpak list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		if pkg := parseFlatpakLine(scanner.Text()); pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read flatpak output: %w", err)
	}

	return packages, nil
}

// parseFlatpakLine parses a single tab-separated line of flatpak list output.
// Columns: name, application ID, version, branch, origin, arch
func parseFlatpakLine(line string) *Package {
	if line == "" {
		return nil
	}

	fields := strings.Split(line, "\t")
	if len(fields) < 2 {
		return nil
	}

	tabField := func(i int) string {
		if i < len(fields) {
			return strings.TrimSpace(fields[i])
		}
		return ""
	}

	name := tabField(0)
	appID := tabField(1)
	version := tabField(2)
	branch := tabField(3)
	origin := tabField(4)
	arch := tabField(5)

	if name == "" {
		name = appID
	}
	if name == "" {
		return nil
	}
	if version == "" {
		version = branch
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   arch,
		PackageManager: "flatpak",
		Source:         origin,
		Description:    appID,
	}
}
