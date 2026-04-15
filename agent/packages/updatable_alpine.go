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

const apkCheckTimeout = 60 * time.Second

// ApkUpdateChecker checks for available updates via apk (Alpine Linux)
type ApkUpdateChecker struct {
	apkPath string
}

func (a *ApkUpdateChecker) Name() string { return "apk-outdated" }

func (a *ApkUpdateChecker) IsAvailable() bool {
	path, err := exec.LookPath("apk")
	if err != nil {
		return false
	}
	a.apkPath = path
	return true
}

func (a *ApkUpdateChecker) PackageManagers() []string {
	return []string{"apk"}
}

func (a *ApkUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apkCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, a.apkPath, "version", "-l", "<")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("apk version failed: %w", err)
	}

	updates := make(map[string]UpdateStatus)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		// Skip the header line: "Installed:      Available:"
		if strings.HasPrefix(line, "Installed:") {
			continue
		}
		name, status, ok := parseApkVersionLine(line)
		if ok {
			updates[name] = status
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan apk output: %w", err)
	}

	return updates, nil
}

// parseApkVersionLine parses a line like:
// "curl-8.11.0-r0         < 8.11.1-r0"
// Alpine has no separate security repository, so HasSecurityUpdate is always false.
func parseApkVersionLine(line string) (string, UpdateStatus, bool) {
	parts := strings.Fields(line)
	if len(parts) < 3 || parts[1] != "<" {
		return "", UpdateStatus{}, false
	}

	name := extractApkPackageName(parts[0])
	if name == "" {
		return "", UpdateStatus{}, false
	}

	return name, UpdateStatus{
		AvailableVersion:  parts[2],
		HasSecurityUpdate: false,
	}, true
}

// extractApkPackageName extracts the package name from "name-version-release".
// The version starts at the first dash-separated component that begins with a digit.
// Examples: "curl-8.11.0-r0" → "curl", "libssl3-3.3.2-r0" → "libssl3"
func extractApkPackageName(nameVersion string) string {
	parts := strings.Split(nameVersion, "-")
	for i, part := range parts {
		if i > 0 && len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
			return strings.Join(parts[:i], "-")
		}
	}
	return ""
}
