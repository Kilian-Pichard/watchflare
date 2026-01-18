package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SnapCollector collects Snap packages (Ubuntu and other distributions)
type SnapCollector struct{}

// Name returns the collector name
func (s *SnapCollector) Name() string {
	return "snap"
}

// IsAvailable checks if snap is available
func (s *SnapCollector) IsAvailable() bool {
	_, err := exec.LookPath("snap")
	return err == nil
}

// Collect gathers all installed snap packages
func (s *SnapCollector) Collect() ([]*Package, error) {
	// Run snap list to get installed packages
	// Output format: Name  Version  Rev   Tracking       Publisher   Notes
	cmd := exec.Command("snap", "list")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("snap list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header line
	firstLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if firstLine {
			firstLine = false
			continue
		}

		// Parse snap list output (space-separated)
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
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

		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			Architecture:   "",        // Snap doesn't expose arch in list
			PackageManager: "snap",
			Source:         channel,   // Use tracking channel as source
			InstalledAt:    time.Time{}, // Not easily available
			PackageSize:    0,          // Would need snap info for each package
			Description:    fmt.Sprintf("Snap package from %s (rev: %s)", publisher, revision),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading snap output: %w", err)
	}

	return packages, nil
}
