package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const checkupdatesTimeout = 120 * time.Second

// PacmanUpdateChecker checks for available updates via checkupdates (Arch Linux).
// checkupdates is provided by the pacman-contrib package and is optional.
// If not installed, IsAvailable() returns false and the checker is skipped.
type PacmanUpdateChecker struct {
	checkupdatesPath string
}

func (p *PacmanUpdateChecker) Name() string { return "checkupdates" }

func (p *PacmanUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("checkupdates")
	if err != nil {
		return false
	}
	p.checkupdatesPath = path
	return true
}

func (p *PacmanUpdateChecker) PackageManagers() []string {
	return []string{"pacman"}
}

func (p *PacmanUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), checkupdatesTimeout)
	defer cancel()

	// checkupdates exits with code 2 when no updates are available — this is not an error
	cmd := exec.CommandContext(ctx, p.checkupdatesPath)
	output, _ := cmd.Output()

	updates := make(map[string]UpdateStatus)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		name, status, ok := parsePacmanUpdateLine(scanner.Text())
		if ok {
			updates[name] = status
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan checkupdates output: %w", err)
	}

	return updates, nil
}

// parsePacmanUpdateLine parses a line like:
// "curl 8.10.1-1 -> 8.11.1-1"
// Arch Linux has no separate security repository, so HasSecurityUpdate is always false.
func parsePacmanUpdateLine(line string) (string, UpdateStatus, bool) {
	parts := strings.Fields(line)
	if len(parts) < 4 || parts[2] != "->" {
		return "", UpdateStatus{}, false
	}
	return parts[0], UpdateStatus{
		AvailableVersion:  parts[3],
		HasSecurityUpdate: false,
	}, true
}
