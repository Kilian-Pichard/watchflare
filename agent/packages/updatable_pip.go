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

const pipOutdatedTimeout = 60 * time.Second

// PipUpdateChecker checks for available updates via pip (cross-platform)
type PipUpdateChecker struct {
	pipPath string
}

func (p *PipUpdateChecker) Name() string { return "pip" }

func (p *PipUpdateChecker) IsAvailable() bool {
	for _, pipCmd := range []string{"pip3", "pip"} {
		path, err := exec.LookPath(pipCmd)
		if err == nil {
			p.pipPath = path
			return true
		}
	}
	return false
}

func (p *PipUpdateChecker) PackageManagers() []string {
	return []string{"pip"}
}

// CheckUpdates returns available updates for installed pip packages.
// Uses "pip list --outdated --format=json".
// Security update detection is not available for pip packages.
func (p *PipUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pipOutdatedTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.pipPath, "list", "--outdated", "--format=json")
	cmd.Env = pipEnvWithDirs()
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip list --outdated failed: %w", err)
	}

	if len(output) == 0 {
		return make(map[string]UpdateStatus), nil
	}

	return parsePipOutdated(output)
}

// pipEnvWithDirs returns the environment for pip commands with XDG_CACHE_HOME
// redirected to /tmp. When the service user has a non-writable home (e.g. /var/empty),
// pip warns about ~/.cache/pip not being writable. Redirecting XDG_CACHE_HOME to a
// writable temp directory prevents this.
func pipEnvWithDirs() []string {
	const tmpDir = "/tmp/watchflare-pip"
	_ = os.MkdirAll(tmpDir, 0700)

	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "XDG_CACHE_HOME=") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "XDG_CACHE_HOME="+tmpDir)
}

// pipOutdatedEntry represents one entry in pip list --outdated --format=json output.
type pipOutdatedEntry struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	LatestVersion string `json:"latest_version"`
}

// parsePipOutdated parses the JSON output of "pip list --outdated --format=json".
// The output is an array of {name, version, latest_version, latest_filetype}.
func parsePipOutdated(output []byte) (map[string]UpdateStatus, error) {
	var entries []pipOutdatedEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse pip outdated JSON: %w", err)
	}

	updates := make(map[string]UpdateStatus, len(entries))
	for _, e := range entries {
		if e.LatestVersion != "" && e.LatestVersion != e.Version {
			updates[e.Name] = UpdateStatus{AvailableVersion: e.LatestVersion}
		}
	}
	return updates, nil
}
