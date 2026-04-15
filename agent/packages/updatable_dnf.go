package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

const dnfCheckTimeout = 120 * time.Second

// DnfUpdateChecker checks for available updates via dnf (RHEL/CentOS/Rocky/Fedora)
type DnfUpdateChecker struct {
	dnfPath string
}

func (d *DnfUpdateChecker) Name() string { return "dnf" }

func (d *DnfUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("dnf")
	if err != nil {
		return false
	}
	d.dnfPath = path
	return true
}

func (d *DnfUpdateChecker) PackageManagers() []string {
	return []string{"rpm"}
}

func (d *DnfUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	updates, err := d.getAllUpdates()
	if err != nil {
		return nil, err
	}

	if err := d.markSecurityUpdates(updates); err != nil {
		// Non-fatal: return all updates without the security flag
		slog.Warn("dnf security info unavailable", "error", err)
	}

	return updates, nil
}

// getAllUpdates returns all available updates via dnf check-update.
// dnf check-update exits with code 100 when updates are available — this is not an error.
// --quiet is intentionally omitted: DNF5 suppresses the package list with --quiet,
// relying solely on the exit code. Non-package lines are filtered in the parser instead.
func (d *DnfUpdateChecker) getAllUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dnfCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, d.dnfPath, "check-update")
	// DNF5 writes to XDG_STATE_HOME and XDG_CACHE_HOME. When the service user has a
	// non-writable home (e.g. /var/empty), these default to ~/.local/state and ~/.cache
	// which fail. Redirect both to a writable temp directory.
	cmd.Env = dnfEnvWithDirs()
	output, err := cmd.Output()
	if err != nil {
		// Exit code 100 means updates are available — not a real error.
		// Any other non-zero exit code is a real failure.
		exitErr, isExitErr := err.(*exec.ExitError)
		if !isExitErr {
			return nil, fmt.Errorf("dnf check-update failed: %w", err)
		}
		if exitErr.ExitCode() != 100 {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			return nil, fmt.Errorf("dnf check-update failed (exit %d): %s", exitErr.ExitCode(), stderr)
		}
	}

	updates := make(map[string]UpdateStatus)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		name, version, ok := parseDnfCheckUpdateLine(line)
		if ok {
			updates[name] = UpdateStatus{AvailableVersion: version}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan dnf output: %w", err)
	}

	return updates, nil
}

// parseDnfCheckUpdateLine parses a line like:
// "curl.x86_64   8.11.1-1.fc40   updates"
// Package lines always have "name.arch" as the first field (contains a dot).
// Header and text lines ("Updating and loading repositories:", "Obsoleting packages", etc.) are skipped.
func parseDnfCheckUpdateLine(line string) (string, string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", false
	}
	// Package lines have "name.arch" format; non-package lines (headers, section titles) don't.
	if !strings.Contains(fields[0], ".") {
		return "", "", false
	}
	nameArch := strings.SplitN(fields[0], ".", 2)
	return nameArch[0], fields[1], true
}

// markSecurityUpdates uses "dnf repoquery --upgrades --security" to flag packages
// with security advisories. repoquery is a pure query command that always writes to
// stdout regardless of TTY status, avoiding the output-suppression issues of
// "dnf updateinfo list" when run via exec.Command (non-TTY pipe).
// The --qf flag outputs only the package name, one per line — no NEVRA parsing needed.
// This command works identically on DNF4 and DNF5.
func (d *DnfUpdateChecker) markSecurityUpdates(updates map[string]UpdateStatus) error {
	if len(updates) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), dnfCheckTimeout)
	defer cancel()

	secCmd := exec.CommandContext(ctx, d.dnfPath,
		"repoquery", "--upgrades", "--security", "--qf", "%{name}\\n",
	)
	secCmd.Env = dnfEnvWithDirs()
	output, err := secCmd.Output()
	if err != nil {
		return fmt.Errorf("dnf repoquery failed: %w", err)
	}

	marked := 0
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}
		if u, ok := updates[name]; ok {
			u.HasSecurityUpdate = true
			updates[name] = u
			marked++
		}
	}

	slog.Debug("dnf security packages marked", "count", marked)
	return scanner.Err()
}

// dnfEnvWithDirs returns the environment for dnf commands with XDG dirs redirected to /tmp.
// DNF5 writes to XDG_STATE_HOME and XDG_CACHE_HOME. When the service user has a
// non-writable home directory (e.g. /var/empty), dnf fails. We redirect these to
// /tmp so dnf can run as any unprivileged user.
func dnfEnvWithDirs() []string {
	const tmpDir = "/tmp/watchflare-dnf"
	// Pre-create directories so DNF5 doesn't fail on first run.
	_ = os.MkdirAll(tmpDir+"/state", 0700)
	_ = os.MkdirAll(tmpDir+"/cache", 0700)

	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		// Drop any existing XDG overrides so our values take effect.
		if strings.HasPrefix(e, "XDG_STATE_HOME=") || strings.HasPrefix(e, "XDG_CACHE_HOME=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env,
		"XDG_STATE_HOME="+tmpDir+"/state",
		"XDG_CACHE_HOME="+tmpDir+"/cache",
	)
	return env
}
