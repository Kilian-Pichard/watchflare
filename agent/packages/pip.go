package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const pipTimeout = 30 * time.Second

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

// Collect gathers all installed pip packages.
// Uses "pip list --format=json".
func (p *PipCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pipTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.pipPath, "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip list failed: %w", err)
	}

	return parsePipOutput(output)
}

// parsePipOutput parses the JSON output of "pip list --format=json".
func parsePipOutput(output []byte) ([]*Package, error) {
	var pipPackages []pipPackage
	if err := json.Unmarshal(output, &pipPackages); err != nil {
		return nil, fmt.Errorf("failed to parse pip JSON: %w", err)
	}

	packages := make([]*Package, 0, len(pipPackages))
	for _, pkg := range pipPackages {
		packages = append(packages, &Package{
			Name:           pkg.Name,
			Version:        pkg.Version,
			PackageManager: "pip",
			Source:         "pypi.org",
		})
	}

	return packages, nil
}
