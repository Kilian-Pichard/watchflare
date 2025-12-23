package packages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BrewCollector collects packages from Homebrew (macOS)
type BrewCollector struct {
	brewPath string
}

// Name returns the collector name
func (b *BrewCollector) Name() string {
	return "brew"
}

// IsAvailable checks if brew is available
func (b *BrewCollector) IsAvailable() bool {
	// Common brew locations (Apple Silicon and Intel Macs)
	brewPaths := []string{
		"/opt/homebrew/bin/brew", // Apple Silicon
		"/usr/local/bin/brew",    // Intel Macs
	}

	for _, path := range brewPaths {
		if _, err := os.Stat(path); err == nil {
			b.brewPath = path
			return true
		}
	}

	return false
}

// brewPackage represents the JSON structure from brew info --json
type brewPackage struct {
	Name        string       `json:"name"`
	Versions    brewVersions `json:"versions"`
	Description string       `json:"desc"`
}

// brewVersions represents the versions field in brew JSON
type brewVersions struct {
	Stable string      `json:"stable"`
	Head   string      `json:"head"`
	Bottle interface{} `json:"bottle"` // Can be bool or object
}

// Collect gathers all installed packages from Homebrew
func (b *BrewCollector) Collect() ([]*Package, error) {
	// Get list of installed formulae
	cmd := exec.Command(b.brewPath, "list", "--formula")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("brew list failed: %w (output: %s)", err, string(output))
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		packageName := strings.TrimSpace(scanner.Text())
		if packageName == "" {
			continue
		}

		// Get package info (version, description)
		pkg, err := b.getPackageInfo(packageName)
		if err != nil {
			// Skip packages that fail to query (e.g., casks, invalid formulae)
			continue
		}

		packages = append(packages, pkg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse brew output: %w", err)
	}

	return packages, nil
}

// getPackageInfo retrieves detailed information for a single package
func (b *BrewCollector) getPackageInfo(name string) (*Package, error) {
	// Get JSON info for the package
	cmd := exec.Command(b.brewPath, "info", "--json=v2", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("brew info failed for %s: %w (output: %s)", name, err, string(output))
	}

	// Parse JSON response
	var response struct {
		Formulae []brewPackage `json:"formulae"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse brew JSON: %w", err)
	}

	if len(response.Formulae) == 0 {
		return nil, fmt.Errorf("no formula found for %s", name)
	}

	formula := response.Formulae[0]

	// Get installed version (stable version)
	version := formula.Versions.Stable
	if version == "" {
		version = "unknown"
	}

	// Truncate description to 100 chars
	description := TruncateDescription(formula.Description)

	return &Package{
		Name:           formula.Name,
		Version:        version,
		Architecture:   "", // Homebrew doesn't provide arch in formula
		PackageManager: "brew",
		Source:         "homebrew/core", // Default tap
		InstalledAt:    time.Time{},     // Homebrew doesn't easily provide install date
		PackageSize:    0,                // Homebrew doesn't easily provide size
		Description:    description,
	}, nil
}
