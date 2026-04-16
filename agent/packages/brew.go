package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const brewTimeout = 60 * time.Second

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
	// Common brew locations
	brewPaths := []string{
		"/opt/homebrew/bin/brew",              // Apple Silicon
		"/usr/local/bin/brew",                 // Intel Macs
		"/home/linuxbrew/.linuxbrew/bin/brew", // Linux
	}

	for _, path := range brewPaths {
		if _, err := os.Stat(path); err == nil {
			b.brewPath = path
			return true
		}
	}

	return false
}

// Collect gathers all installed Homebrew formulae and casks in two calls.
func (b *BrewCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), brewTimeout)
	defer cancel()

	var pkgs []*Package

	formulae, err := b.collectFormulae(ctx)
	if err != nil {
		return nil, err
	}
	pkgs = append(pkgs, formulae...)

	casks, err := b.collectCasks(ctx)
	if err != nil {
		// Casks are optional — don't fail the whole collection
		return pkgs, nil
	}
	pkgs = append(pkgs, casks...)

	return pkgs, nil
}

// collectFormulae fetches all installed formulae in a single brew call.
func (b *BrewCollector) collectFormulae(ctx context.Context) ([]*Package, error) {
	cmd := exec.CommandContext(ctx, b.brewPath, "info", "--json=v2", "--formula", "--installed")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("brew info --formula --installed failed: %w", err)
	}
	return parseBrewFormulaeJSON(output)
}

// collectCasks fetches all installed casks in a single brew call.
func (b *BrewCollector) collectCasks(ctx context.Context) ([]*Package, error) {
	cmd := exec.CommandContext(ctx, b.brewPath, "info", "--json=v2", "--cask", "--installed")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("brew info --cask --installed failed: %w", err)
	}
	return parseBrewCasksJSON(output)
}

// parseBrewFormulaeJSON parses the JSON output of "brew info --json=v2 --formula --installed".
func parseBrewFormulaeJSON(output []byte) ([]*Package, error) {
	var response struct {
		Formulae []struct {
			Name      string `json:"name"`
			FullName  string `json:"full_name"`
			Desc      string `json:"desc"`
			Installed []struct {
				Version string `json:"version"`
			} `json:"installed"`
		} `json:"formulae"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse brew formula JSON: %w", err)
	}

	pkgs := make([]*Package, 0, len(response.Formulae))
	for _, f := range response.Formulae {
		version := "unknown"
		if len(f.Installed) > 0 && f.Installed[0].Version != "" {
			version = f.Installed[0].Version
		}
		name := f.FullName
		if name == "" {
			name = f.Name
		}
		pkgs = append(pkgs, &Package{
			Name:           name,
			Version:        version,
			PackageManager: "brew-formula",
			Source:         "homebrew/core",
			Description:    TruncateDescription(f.Desc),
		})
	}

	return pkgs, nil
}

// parseBrewCasksJSON parses the JSON output of "brew info --json=v2 --cask --installed".
func parseBrewCasksJSON(output []byte) ([]*Package, error) {
	var response struct {
		Casks []struct {
			Token     string `json:"token"`
			Installed string `json:"installed"` // actually installed version
			Desc      string `json:"desc"`
		} `json:"casks"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse brew cask JSON: %w", err)
	}

	pkgs := make([]*Package, 0, len(response.Casks))
	for _, c := range response.Casks {
		version := c.Installed
		if version == "" {
			version = "unknown"
		}
		pkgs = append(pkgs, &Package{
			Name:           c.Token,
			Version:        version,
			PackageManager: "brew-cask",
			Source:         "homebrew/cask",
			Description:    TruncateDescription(c.Desc),
		})
	}

	return pkgs, nil
}
