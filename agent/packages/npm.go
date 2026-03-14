package packages

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

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

// npmPackage represents the JSON structure from npm list
type npmPackage struct {
	Version      string                 `json:"version"`
	Dependencies map[string]npmPackage  `json:"dependencies"`
}

// npmListOutput represents the JSON output from npm list --global --json
type npmListOutput struct {
	Version      string                 `json:"version"`
	Name         string                 `json:"name"`
	Dependencies map[string]npmPackage  `json:"dependencies"`
}

// Collect gathers all globally installed npm packages
func (n *NpmCollector) Collect() ([]*Package, error) {
	// Run npm list --global --json --depth=0 to get global packages
	cmd := exec.Command(n.npmPath, "list", "--global", "--json", "--depth=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// npm list returns exit code 1 even on success if there are any issues
		// So we check if we got valid JSON output
		if len(output) == 0 {
			return nil, fmt.Errorf("npm list failed: %w (output: %s)", err, string(output))
		}
	}

	// Parse JSON response
	var listOutput npmListOutput
	if err := json.Unmarshal(output, &listOutput); err != nil {
		return nil, fmt.Errorf("failed to parse npm JSON: %w", err)
	}

	var packages []*Package

	// Collect all globally installed packages (including npm itself)
	for name, pkg := range listOutput.Dependencies {
		packages = append(packages, &Package{
			Name:           name,
			Version:        pkg.Version,
			Architecture:   "",        // npm is platform-independent
			PackageManager: "npm",
			Source:         "npmjs.com", // Default npm registry
			InstalledAt:    time.Time{}, // npm doesn't easily provide install date
			PackageSize:    0,            // Would require scanning node_modules
			Description:    "",           // Would require additional npm info call per package
		})
	}

	return packages, nil
}
