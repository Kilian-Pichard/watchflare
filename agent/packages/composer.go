package packages

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ComposerCollector collects globally installed Composer packages (cross-platform)
type ComposerCollector struct {
	composerPath string
}

// Name returns the collector name
func (c *ComposerCollector) Name() string {
	return "composer"
}

// IsAvailable checks if composer is available
func (c *ComposerCollector) IsAvailable() bool {
	composerPath, err := exec.LookPath("composer")
	if err != nil {
		return false
	}

	c.composerPath = composerPath
	return true
}

// composerPackage represents a package from composer global show --format=json
type composerPackage struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// composerGlobalOutput represents the JSON output from composer global show
type composerGlobalOutput struct {
	Installed []composerPackage `json:"installed"`
}

// Collect gathers all globally installed Composer packages
func (c *ComposerCollector) Collect() ([]*Package, error) {
	// Run composer global show --format=json
	cmd := exec.Command(c.composerPath, "global", "show", "--format=json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If no global packages are installed, composer might return an error
		if len(output) == 0 {
			return []*Package{}, nil
		}
	}

	// Parse JSON response
	var globalOutput composerGlobalOutput
	if err := json.Unmarshal(output, &globalOutput); err != nil {
		return nil, fmt.Errorf("failed to parse composer JSON: %w", err)
	}

	var packages []*Package

	for _, pkg := range globalOutput.Installed {
		// Truncate description to 100 chars
		description := TruncateDescription(pkg.Description)

		packages = append(packages, &Package{
			Name:           pkg.Name,
			Version:        pkg.Version,
			Architecture:   "",             // composer is platform-independent
			PackageManager: "composer",
			Source:         "packagist.org", // Default Composer registry
			InstalledAt:    time.Time{},    // composer doesn't easily provide install date
			PackageSize:    0,               // Would require scanning vendor directory
			Description:    description,
		})
	}

	return packages, nil
}
