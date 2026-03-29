package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const composerTimeout = 30 * time.Second

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
	ctx, cancel := context.WithTimeout(context.Background(), composerTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.composerPath, "global", "show", "--format=json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Composer exits non-zero when no global packages are installed
		if len(output) == 0 {
			return []*Package{}, nil
		}
		return nil, fmt.Errorf("composer global show failed: %w (output: %s)", err, string(output))
	}

	return parseComposerJSON(output)
}

// parseComposerJSON parses the JSON output of "composer global show --format=json".
func parseComposerJSON(output []byte) ([]*Package, error) {
	var globalOutput composerGlobalOutput
	if err := json.Unmarshal(output, &globalOutput); err != nil {
		return nil, fmt.Errorf("failed to parse composer JSON: %w", err)
	}

	packages := make([]*Package, 0, len(globalOutput.Installed))
	for _, pkg := range globalOutput.Installed {
		packages = append(packages, &Package{
			Name:           pkg.Name,
			Version:        pkg.Version,
			PackageManager: "composer",
			Source:         "packagist.org",
			Description:    TruncateDescription(pkg.Description),
		})
	}

	return packages, nil
}
