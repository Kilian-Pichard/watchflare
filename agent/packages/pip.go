package packages

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// PipCollector collects installed pip packages (cross-platform)
type PipCollector struct {
	pipPath string
}

// Name returns the collector name
func (p *PipCollector) Name() string {
	return "pip"
}

// IsAvailable checks if pip is available
func (p *PipCollector) IsAvailable() bool {
	// Try pip3 first (more common on modern systems)
	for _, pipCmd := range []string{"pip3", "pip"} {
		pipPath, err := exec.LookPath(pipCmd)
		if err == nil {
			p.pipPath = pipPath
			return true
		}
	}

	return false
}

// pipPackage represents a package from pip list --format=json
type pipPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Collect gathers all installed pip packages
func (p *PipCollector) Collect() ([]*Package, error) {
	// Run pip list --format=json
	// Use Output() instead of CombinedOutput() to avoid stderr warnings
	cmd := exec.Command(p.pipPath, "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip list failed: %w", err)
	}

	// Parse JSON response
	var pipPackages []pipPackage
	if err := json.Unmarshal(output, &pipPackages); err != nil {
		return nil, fmt.Errorf("failed to parse pip JSON: %w", err)
	}

	var packages []*Package

	for _, pkg := range pipPackages {
		packages = append(packages, &Package{
			Name:           pkg.Name,
			Version:        pkg.Version,
			Architecture:   "",        // pip is platform-independent
			PackageManager: "pip",
			Source:         "pypi.org", // Default PyPI registry
			InstalledAt:    time.Time{}, // pip doesn't easily provide install date
			PackageSize:    0,            // Would require scanning site-packages
			Description:    "",           // Would require additional pip show call per package
		})
	}

	return packages, nil
}
