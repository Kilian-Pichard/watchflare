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
	var pkgs []*Package

	// Collect formulae
	formulae, err := b.collectFormulae()
	if err != nil {
		return nil, err
	}
	pkgs = append(pkgs, formulae...)

	// Collect casks
	casks, err := b.collectCasks()
	if err != nil {
		// Don't fail if cask collection fails, just log warning
		// (casks might not be available on all systems)
		return pkgs, nil
	}
	pkgs = append(pkgs, casks...)

	return pkgs, nil
}

// collectFormulae collects installed Homebrew formulae
func (b *BrewCollector) collectFormulae() ([]*Package, error) {
	cmd := exec.Command(b.brewPath, "list", "--formula")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("brew list --formula failed: %w (output: %s)", err, string(output))
	}

	var pkgs []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		packageName := strings.TrimSpace(scanner.Text())
		if packageName == "" {
			continue
		}

		// Get package info (version, description)
		pkg, err := b.getPackageInfo(packageName, "formula")
		if err != nil {
			// Skip packages that fail to query
			continue
		}

		pkgs = append(pkgs, pkg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse brew formula output: %w", err)
	}

	return pkgs, nil
}

// collectCasks collects installed Homebrew casks
func (b *BrewCollector) collectCasks() ([]*Package, error) {
	cmd := exec.Command(b.brewPath, "list", "--cask")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("brew list --cask failed: %w", err)
	}

	var pkgs []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		caskName := strings.TrimSpace(scanner.Text())
		if caskName == "" {
			continue
		}

		// Get cask info (version, description)
		pkg, err := b.getCaskInfo(caskName)
		if err != nil {
			// Skip casks that fail to query
			continue
		}

		pkgs = append(pkgs, pkg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse brew cask output: %w", err)
	}

	return pkgs, nil
}

// getPackageInfo retrieves detailed information for a single formula
func (b *BrewCollector) getPackageInfo(name string, packageType string) (*Package, error) {
	// Get JSON info for the package with --formula flag to avoid ambiguity
	cmd := exec.Command(b.brewPath, "info", "--json=v2", "--formula", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("brew info --formula failed for %s: %w (output: %s)", name, err, string(output))
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
		PackageManager: "brew-formulae",
		Source:         "homebrew/core", // Default tap
		InstalledAt:    time.Time{},     // Homebrew doesn't easily provide install date
		PackageSize:    0,                // Homebrew doesn't easily provide size
		Description:    description,
	}, nil
}

// getCaskInfo retrieves detailed information for a single cask
func (b *BrewCollector) getCaskInfo(name string) (*Package, error) {
	// Get JSON info for the cask
	cmd := exec.Command(b.brewPath, "info", "--json=v2", "--cask", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("brew info --cask failed for %s: %w", name, err)
	}

	// Parse JSON response for casks
	var response struct {
		Casks []struct {
			Token       string   `json:"token"`
			Version     string   `json:"version"`
			Description string   `json:"desc"`
			Homepage    string   `json:"homepage"`
		} `json:"casks"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse brew cask JSON: %w", err)
	}

	if len(response.Casks) == 0 {
		return nil, fmt.Errorf("no cask found for %s", name)
	}

	cask := response.Casks[0]

	// Truncate description to 100 chars
	description := TruncateDescription(cask.Description)

	return &Package{
		Name:           cask.Token,
		Version:        cask.Version,
		Architecture:   "", // Casks are typically multi-arch
		PackageManager: "brew-casks",
		Source:         "homebrew/cask", // Cask tap
		InstalledAt:    time.Time{},
		PackageSize:    0,
		Description:    description,
	}, nil
}
