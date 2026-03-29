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

const snapTimeout = 30 * time.Second

// SnapCollector collects Snap packages (Ubuntu and other distributions)
type SnapCollector struct {
	snapPath string
}

// Name returns the collector name
func (s *SnapCollector) Name() string {
	return "snap"
}

// IsAvailable checks if snap is available
func (s *SnapCollector) IsAvailable() bool {
	path, err := exec.LookPath("snap")
	if err != nil {
		return false
	}
	s.snapPath = path
	return true
}

// Collect gathers all installed snap packages.
// Uses "snap list". Output format: Name  Version  Rev  Tracking  Publisher  Notes
func (s *SnapCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), snapTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.snapPath, "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("snap list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	headerPassed := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !headerPassed {
			headerPassed = true
			continue
		}
		if pkg := parseSnapLine(line); pkg != nil {
			packages = append(packages, pkg)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading snap output: %w", err)
	}

	return packages, nil
}

// parseSnapLine parses a single data line of "snap list" output.
// Format: "name  version  rev  tracking  publisher  notes"
func parseSnapLine(line string) *Package {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	name := fields[0]
	version := fields[1]
	revision := ""
	channel := ""
	publisher := ""

	if len(fields) >= 3 {
		revision = fields[2]
	}
	if len(fields) >= 4 {
		channel = fields[3]
	}
	if len(fields) >= 5 {
		publisher = fields[4]
	}

	var description string
	switch {
	case publisher != "" && revision != "":
		description = fmt.Sprintf("Snap package from %s (rev: %s)", publisher, revision)
	case publisher != "":
		description = fmt.Sprintf("Snap package from %s", publisher)
	case revision != "":
		description = fmt.Sprintf("rev: %s", revision)
	}

	return &Package{
		Name:           name,
		Version:        version,
		PackageManager: "snap",
		Source:         channel,
		Description:    description,
	}
}
