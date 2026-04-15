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

const npmOutdatedTimeout = 60 * time.Second

// NpmUpdateChecker checks for available updates via npm (cross-platform)
type NpmUpdateChecker struct {
	npmPath string
}

func (n *NpmUpdateChecker) Name() string { return "npm" }

func (n *NpmUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("npm")
	if err != nil {
		return false
	}
	n.npmPath = path
	return true
}

func (n *NpmUpdateChecker) PackageManagers() []string {
	return []string{"npm"}
}

// CheckUpdates returns available updates for globally installed npm packages.
// Uses "npm outdated -g --json". npm exits with code 1 when outdated packages
// are found — this is not an error.
// Security update detection is not available for global npm packages.
func (n *NpmUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), npmOutdatedTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, n.npmPath, "outdated", "-g", "--json")
	cmd.Env = npmEnvWithDirs()
	output, err := cmd.Output()
	if err != nil {
		exitErr, isExitErr := err.(*exec.ExitError)
		if !isExitErr {
			return nil, fmt.Errorf("npm outdated failed: %w", err)
		}
		// Exit code 1 means outdated packages found — not a real error.
		if exitErr.ExitCode() != 1 {
			return nil, fmt.Errorf("npm outdated failed (exit %d): %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
	}

	if len(output) == 0 {
		return make(map[string]UpdateStatus), nil
	}

	return parseNpmOutdated(output)
}

// npmEnvWithDirs returns the environment for npm commands with HOME and cache
// redirected to /tmp. When the service user has a non-writable home (e.g. /var/empty),
// npm fails trying to create ~/.npm. Redirecting HOME and npm_config_cache to a
// writable temp directory prevents this.
func npmEnvWithDirs() []string {
	const tmpDir = "/tmp/watchflare-npm"
	_ = os.MkdirAll(tmpDir, 0700)

	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "npm_config_cache=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env,
		"HOME="+tmpDir,
		"npm_config_cache="+tmpDir,
	)
	return env
}

// npmOutdatedEntry represents one entry in npm outdated -g --json output.
type npmOutdatedEntry struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// parseNpmOutdated parses the JSON output of "npm outdated -g --json".
// The output is a map of package name → {current, wanted, latest, ...}.
func parseNpmOutdated(output []byte) (map[string]UpdateStatus, error) {
	var raw map[string]npmOutdatedEntry
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse npm outdated JSON: %w", err)
	}

	updates := make(map[string]UpdateStatus, len(raw))
	for name, entry := range raw {
		if entry.Latest != "" && entry.Latest != entry.Current {
			updates[name] = UpdateStatus{AvailableVersion: entry.Latest}
		}
	}
	return updates, nil
}
