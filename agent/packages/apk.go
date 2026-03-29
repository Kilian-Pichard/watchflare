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

const apkTimeout = 60 * time.Second

// ApkCollector collects packages from apk (Alpine Linux)
type ApkCollector struct {
	apkPath string
}

// Name returns the collector name
func (a *ApkCollector) Name() string {
	return "apk"
}

// IsAvailable checks if apk is available
func (a *ApkCollector) IsAvailable() bool {
	path, err := exec.LookPath("apk")
	if err != nil {
		return false
	}
	a.apkPath = path
	return true
}

// Collect gathers all installed packages from apk.
// Uses a single "apk info -vv" call which returns all metadata per package
// in multi-line blocks separated by blank lines.
func (a *ApkCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apkTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, a.apkPath, "info", "-vv")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("apk info failed: %w", err)
	}

	var packages []*Package
	var block []string

	flush := func() {
		if len(block) == 0 {
			return
		}
		if pkg := parseApkBlock(block); pkg != nil {
			packages = append(packages, pkg)
		}
		block = block[:0]
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
		} else {
			block = append(block, line)
		}
	}
	flush() // last block (no trailing blank line)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading apk output: %w", err)
	}

	return packages, nil
}

// parseApkBlock parses a multi-line apk info -vv block for one package.
//
// Example block:
//
//	musl-1.2.4-r2 description:
//	A C library designed for Linux
//	musl-1.2.4-r2 webpage:
//	https://musl.libc.org/
//	musl-1.2.4-r2 installed size:
//	624 KiB
//	musl-1.2.4-r2 arch:
//	x86_64
func parseApkBlock(block []string) *Package {
	if len(block) == 0 {
		return nil
	}

	// First line identifies the package: "<name>-<version> description:" or similar
	firstLine := block[0]
	spaceIdx := strings.Index(firstLine, " ")
	if spaceIdx < 0 {
		return nil
	}
	fullName := firstLine[:spaceIdx]

	name, version := splitNameVersion(fullName)
	if name == "" {
		return nil
	}

	pkg := &Package{
		Name:           name,
		Version:        version,
		PackageManager: "apk",
	}

	// Parse key/value pairs: odd lines are keys, even lines are values
	for i := 0; i+1 < len(block); i += 2 {
		key := block[i]
		value := strings.TrimSpace(block[i+1])

		switch {
		case strings.HasSuffix(key, " description:"):
			pkg.Description = TruncateDescription(value)
		case strings.HasSuffix(key, " installed size:"):
			pkg.PackageSize = parseApkSize(value)
		case strings.HasSuffix(key, " arch:"):
			pkg.Architecture = value
		case strings.HasSuffix(key, " provides:"):
			// e.g. repository origin — skip, not directly available here
		}
	}

	return pkg
}

// splitNameVersion splits an apk package string into name and version.
// Alpine packages follow the pattern: <name>-<version>-r<release>
// where the version always starts with a digit.
// Example: "alpine-base-3.18.4-r0" → name="alpine-base", version="3.18.4-r0"
func splitNameVersion(fullName string) (string, string) {
	parts := strings.Split(fullName, "-")
	if len(parts) < 2 {
		return fullName, ""
	}

	// Walk forward: version starts at the first segment that begins with a digit
	// AND is preceded by a non-digit segment (avoids splitting lib2to3-1.0 too early)
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 && parts[i][0] >= '0' && parts[i][0] <= '9' {
			return strings.Join(parts[:i], "-"), strings.Join(parts[i:], "-")
		}
	}

	return fullName, ""
}

// parseApkSize converts apk size strings to bytes.
// Input examples: "624 KiB", "1.2 MiB", "4096"
func parseApkSize(sizeStr string) int64 {
	parts := strings.Fields(strings.TrimSpace(sizeStr))
	if len(parts) == 0 {
		return 0
	}

	// Try plain integer first (bytes)
	var value float64
	_, err := fmt.Sscanf(parts[0], "%f", &value)
	if err != nil || value < 0 {
		return 0
	}

	if len(parts) == 1 {
		return int64(value)
	}

	switch strings.ToLower(parts[1]) {
	case "kib", "k":
		return int64(value * 1024)
	case "mib", "m":
		return int64(value * 1024 * 1024)
	case "gib", "g":
		return int64(value * 1024 * 1024 * 1024)
	}

	return int64(value)
}
