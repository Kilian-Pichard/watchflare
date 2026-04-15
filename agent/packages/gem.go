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

const gemTimeout = 30 * time.Second

// gemLineRegex matches "name (version1, version2, ...)" from gem list output.
var gemLineRegex = regexp.MustCompile(`^(\S+)\s+\(([^)]+)\)`)

// GemCollector collects installed Ruby gems (cross-platform)
type GemCollector struct {
	gemPath string
}

// Name returns the collector name
func (g *GemCollector) Name() string {
	return "gem"
}

// IsAvailable checks if gem is available
func (g *GemCollector) IsAvailable() bool {
	gemPath, err := exec.LookPath("gem")
	if err != nil {
		return false
	}
	g.gemPath = gemPath
	return true
}

// Collect gathers all installed Ruby gems.
// Parses "gem list --local" output:
//
//	bundler (2.4.10, 2.3.26)
func (g *GemCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gemTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, g.gemPath, "list", "--local")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gem list failed: %w (output: %s)", err, string(output))
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		if pkg := parseGemLine(scanner.Text()); pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse gem output: %w", err)
	}

	return packages, nil
}

// parseGemLine parses a single line of gem list output.
// Format: "name (version1, version2, ...)"
// Reports only the first (latest) version when multiple are installed.
func parseGemLine(line string) *Package {
	if strings.TrimSpace(line) == "" {
		return nil
	}

	matches := gemLineRegex.FindStringSubmatch(line)
	if len(matches) < 3 {
		return nil
	}

	name := matches[1]
	version := strings.TrimSpace(strings.SplitN(matches[2], ",", 2)[0])
	// Strip "default: " prefix for gems bundled with Ruby
	version = strings.TrimPrefix(version, "default: ")

	return &Package{
		Name:           name,
		Version:        version,
		PackageManager: "gem",
		Source:         "rubygems.org",
	}
}
