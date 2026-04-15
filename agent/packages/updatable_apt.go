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

const aptCheckTimeout = 60 * time.Second

// AptUpdateChecker checks for available updates via apt (Debian/Ubuntu)
type AptUpdateChecker struct {
	aptPath string
}

func (a *AptUpdateChecker) Name() string { return "apt" }

func (a *AptUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("apt")
	if err != nil {
		return false
	}
	a.aptPath = path
	return true
}

func (a *AptUpdateChecker) PackageManagers() []string {
	return []string{"dpkg"}
}

func (a *AptUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), aptCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, a.aptPath, "list", "--upgradable")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("apt list --upgradable failed: %w", err)
	}

	updates := make(map[string]UpdateStatus)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		// Skip the "Listing..." header line and empty lines
		if strings.HasPrefix(line, "Listing") || line == "" {
			continue
		}
		name, status, ok := parseAptUpgradableLine(line)
		if ok {
			updates[name] = status
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan apt output: %w", err)
	}

	return updates, nil
}

// parseAptUpgradableLine parses a line like:
// "curl/stable-security 8.11.1-1 amd64 [upgradable from: 8.11.0-6]"
// Security updates come from repos whose name contains "security".
func parseAptUpgradableLine(line string) (string, UpdateStatus, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", UpdateStatus{}, false
	}

	// parts[0] is "name/repo", parts[1] is the available version
	nameRepo := strings.SplitN(parts[0], "/", 2)
	if len(nameRepo) < 2 {
		return "", UpdateStatus{}, false
	}

	name := nameRepo[0]
	repo := nameRepo[1]
	isSecurity := strings.Contains(repo, "security")

	return name, UpdateStatus{
		AvailableVersion:  parts[1],
		HasSecurityUpdate: isSecurity,
	}, true
}
