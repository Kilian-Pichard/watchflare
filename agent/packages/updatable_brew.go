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

const brewOutdatedTimeout = 120 * time.Second

// BrewUpdateChecker checks for available updates via brew outdated (macOS/Linux)
type BrewUpdateChecker struct {
	brewPath string
}

func (b *BrewUpdateChecker) Name() string { return "brew-outdated" }

func (b *BrewUpdateChecker) IsAvailable() bool {
	brewPaths := []string{
		"/opt/homebrew/bin/brew", // Apple Silicon
		"/usr/local/bin/brew",    // Intel Macs and Linux
	}
	for _, path := range brewPaths {
		if _, err := os.Stat(path); err == nil {
			b.brewPath = path
			return true
		}
	}
	return false
}

func (b *BrewUpdateChecker) PackageManagers() []string {
	return []string{"brew-formula", "brew-cask"}
}

func (b *BrewUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), brewOutdatedTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, b.brewPath, "outdated", "--json=v2")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("brew outdated failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("brew outdated failed: %w", err)
	}

	return parseBrewOutdatedJSON(output)
}

// parseBrewOutdatedJSON parses the JSON output of "brew outdated --json=v2".
// Homebrew has no security advisory integration, so HasSecurityUpdate is always false.
func parseBrewOutdatedJSON(output []byte) (map[string]UpdateStatus, error) {
	var result struct {
		Formulae []struct {
			Name           string `json:"name"`
			CurrentVersion string `json:"current_version"`
		} `json:"formulae"`
		Casks []struct {
			Name           string `json:"name"`
			CurrentVersion string `json:"current_version"`
		} `json:"casks"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse brew outdated output: %w", err)
	}

	updates := make(map[string]UpdateStatus)

	for _, f := range result.Formulae {
		updates[f.Name] = UpdateStatus{
			AvailableVersion:  f.CurrentVersion,
			HasSecurityUpdate: false,
		}
	}
	for _, c := range result.Casks {
		updates[c.Name] = UpdateStatus{
			AvailableVersion:  c.CurrentVersion,
			HasSecurityUpdate: false,
		}
	}

	return updates, nil
}
