package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const dpkgTimeout = 60 * time.Second

// DpkgCollector collects packages from dpkg (Debian/Ubuntu)
type DpkgCollector struct {
	dpkgPath string
}

// Name returns the collector name
func (d *DpkgCollector) Name() string {
	return "dpkg"
}

// IsAvailable checks if dpkg-query is available
func (d *DpkgCollector) IsAvailable() bool {
	dpkgPath, err := exec.LookPath("dpkg-query")
	if err != nil {
		return false
	}
	d.dpkgPath = dpkgPath
	return true
}

// Collect gathers all installed packages from dpkg
func (d *DpkgCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dpkgTimeout)
	defer cancel()

	// Format: Package|Version|Architecture|Installed-Size|Status|Description
	cmd := exec.CommandContext(ctx, d.dpkgPath, "-W",
		"-f=${Package}|${Version}|${Architecture}|${Installed-Size}|${db:Status-Abbrev}|${Description}\n")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dpkg-query failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		if pkg := parseDpkgLine(scanner.Text()); pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse dpkg output: %w", err)
	}

	return packages, nil
}

// parseDpkgLine parses a single line of dpkg-query output.
// Format: "name|version|arch|size_kb|status|description"
// Returns nil for empty lines, lines with too few fields, or non-installed packages.
func parseDpkgLine(line string) *Package {
	if line == "" {
		return nil
	}

	fields := strings.Split(line, "|")
	if len(fields) < 6 {
		return nil
	}

	status := strings.TrimSpace(fields[4])
	if !strings.HasPrefix(status, "ii") {
		return nil
	}

	return &Package{
		Name:           fields[0],
		Version:        fields[1],
		Architecture:   fields[2],
		PackageManager: "dpkg",
		PackageSize:    parseInt64(fields[3]) * 1024,
		Description:    TruncateDescription(fields[5]),
	}
}

// parseInt64 safely parses a string to int64, returning 0 on error
func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}

	return val
}
