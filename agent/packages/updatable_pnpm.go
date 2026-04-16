package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const pnpmOutdatedTimeout = 60 * time.Second

// PnpmUpdateChecker checks for available updates via pnpm (cross-platform)
type PnpmUpdateChecker struct {
	pnpmPath string
}

func (p *PnpmUpdateChecker) Name() string { return "pnpm" }

func (p *PnpmUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("pnpm")
	if err != nil {
		return false
	}
	p.pnpmPath = path
	return true
}

func (p *PnpmUpdateChecker) PackageManagers() []string {
	return []string{"pnpm-global"}
}

// CheckUpdates returns available updates for globally installed pnpm packages.
// Uses "pnpm outdated -g --format=json".
// Security update detection is not available for pnpm packages.
func (p *PnpmUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pnpmOutdatedTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.pnpmPath, "outdated", "-g", "--format=json")
	cmd.Env = pnpmEnvWithDirs()
	cmd.Dir = "/tmp/watchflare-pnpm"
	output, err := cmd.Output()
	if err != nil {
		return make(map[string]UpdateStatus), nil
	}

	if len(output) == 0 {
		return make(map[string]UpdateStatus), nil
	}

	return parsePnpmOutdated(output)
}

// pnpmOutdatedEntry represents one entry in pnpm outdated -g --format=json output.
type pnpmOutdatedEntry struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// parsePnpmOutdated parses the JSON output of "pnpm outdated -g --format=json".
// The output is a map of package name → {current, latest, wanted, ...}.
func parsePnpmOutdated(output []byte) (map[string]UpdateStatus, error) {
	var raw map[string]pnpmOutdatedEntry
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse pnpm outdated JSON: %w", err)
	}

	updates := make(map[string]UpdateStatus, len(raw))
	for name, entry := range raw {
		if entry.Latest != "" && entry.Latest != entry.Current {
			updates[name] = UpdateStatus{AvailableVersion: entry.Latest}
		}
	}
	return updates, nil
}
