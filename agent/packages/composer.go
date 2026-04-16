package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	cmd.Env = composerEnvWithHome()
	output, err := cmd.Output()
	if err != nil {
		// Composer exits non-zero when no global packages are installed (no composer.json).
		// In that case stdout is empty or non-JSON — return empty list.
		return []*Package{}, nil
	}

	return parseComposerJSON(output)
}

// composerEnvWithHome returns the environment for composer commands with COMPOSER_HOME
// redirected to /tmp. When the service user has a non-writable home (e.g. /var/empty),
// composer fails trying to access ~/.config/composer. Redirecting COMPOSER_HOME to a
// writable temp directory prevents this.
func composerEnvWithHome() []string {
	const tmpDir = "/tmp/watchflare-composer"
	_ = os.MkdirAll(tmpDir, 0700)

	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "COMPOSER_HOME=") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "COMPOSER_HOME="+tmpDir)
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
