package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const gemOutdatedTimeout = 60 * time.Second

// gemOutdatedRegex matches "name (current < latest)" from gem outdated output.
var gemOutdatedRegex = regexp.MustCompile(`^(\S+)\s+\((\S+)\s+<\s+(\S+)\)`)

// GemUpdateChecker checks for available updates via gem (cross-platform)
type GemUpdateChecker struct {
	gemPath string
}

func (g *GemUpdateChecker) Name() string { return "gem" }

func (g *GemUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("gem")
	if err != nil {
		return false
	}
	g.gemPath = path
	return true
}

func (g *GemUpdateChecker) PackageManagers() []string {
	return []string{"gem"}
}

// CheckUpdates returns available updates for installed Ruby gems.
// Uses "gem outdated". Output format: "name (current < latest)".
// Security update detection is not available for gem packages.
func (g *GemUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gemOutdatedTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, g.gemPath, "outdated")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gem outdated failed: %w", err)
	}

	return parseGemOutdated(output), nil
}

// parseGemOutdated parses the output of "gem outdated".
// Each line: "name (current < latest)"
func parseGemOutdated(output []byte) map[string]UpdateStatus {
	updates := make(map[string]UpdateStatus)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := gemOutdatedRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}
		name := matches[1]
		latest := matches[3]
		updates[name] = UpdateStatus{AvailableVersion: latest}
	}

	return updates
}
