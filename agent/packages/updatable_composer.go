package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const composerOutdatedTimeout = 60 * time.Second

// ComposerUpdateChecker checks for available updates via composer (cross-platform)
type ComposerUpdateChecker struct {
	composerPath string
}

func (c *ComposerUpdateChecker) Name() string { return "composer" }

func (c *ComposerUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("composer")
	if err != nil {
		return false
	}
	c.composerPath = path
	return true
}

func (c *ComposerUpdateChecker) PackageManagers() []string {
	return []string{"composer"}
}

// CheckUpdates returns available updates for globally installed Composer packages.
// Uses "composer global outdated --format=json".
// Output format mirrors "composer global show" with an additional "latest" field.
// Security update detection is not available for composer packages.
func (c *ComposerUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), composerOutdatedTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.composerPath, "global", "outdated", "--format=json")
	cmd.Env = composerEnvWithHome()
	output, err := cmd.Output()
	if err != nil {
		// No global packages installed — return empty.
		return make(map[string]UpdateStatus), nil
	}

	return parseComposerOutdated(output)
}

// composerOutdatedPackage represents one entry in composer global outdated --format=json.
type composerOutdatedPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Latest  string `json:"latest"`
}

// composerOutdatedOutput represents the JSON output from composer global outdated.
type composerOutdatedOutput struct {
	Installed []composerOutdatedPackage `json:"installed"`
}

// parseComposerOutdated parses the JSON output of "composer global outdated --format=json".
func parseComposerOutdated(output []byte) (map[string]UpdateStatus, error) {
	var result composerOutdatedOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse composer outdated JSON: %w", err)
	}

	updates := make(map[string]UpdateStatus, len(result.Installed))
	for _, pkg := range result.Installed {
		if pkg.Latest != "" && pkg.Latest != pkg.Version {
			updates[pkg.Name] = UpdateStatus{AvailableVersion: pkg.Latest}
		}
	}
	return updates, nil
}
