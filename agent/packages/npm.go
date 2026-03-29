package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const npmTimeout = 30 * time.Second

// NpmCollector collects globally installed npm packages (cross-platform)
type NpmCollector struct {
	npmPath string
}

// Name returns the collector name
func (n *NpmCollector) Name() string {
	return "npm"
}

// IsAvailable checks if npm is available
func (n *NpmCollector) IsAvailable() bool {
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return false
	}
	n.npmPath = npmPath
	return true
}

// npmListOutput represents the JSON output from npm list --global --json --depth=0
type npmListOutput struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

// Collect gathers all globally installed npm packages.
// Uses "npm list --global --json --depth=0".
func (n *NpmCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), npmTimeout)
	defer cancel()

	// npm list exits with code 1 even on success when there are peer dep warnings,
	// so we only fail if output is empty.
	cmd := exec.CommandContext(ctx, n.npmPath, "list", "--global", "--json", "--depth=0")
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return nil, fmt.Errorf("npm list failed: %w", err)
	}

	return parseNpmOutput(output)
}

// parseNpmOutput parses the JSON output of npm list --global --json --depth=0.
func parseNpmOutput(output []byte) ([]*Package, error) {
	var listOutput npmListOutput
	if err := json.Unmarshal(output, &listOutput); err != nil {
		return nil, fmt.Errorf("failed to parse npm JSON: %w", err)
	}

	packages := make([]*Package, 0, len(listOutput.Dependencies))
	for name, pkg := range listOutput.Dependencies {
		packages = append(packages, &Package{
			Name:           name,
			Version:        pkg.Version,
			PackageManager: "npm",
			Source:         "npmjs.com",
		})
	}

	return packages, nil
}
