package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

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

// Collect gathers all installed Ruby gems
func (g *GemCollector) Collect() ([]*Package, error) {
	// Run gem list --local
	cmd := exec.Command(g.gemPath, "list", "--local")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gem list failed: %w (output: %s)", err, string(output))
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Pattern: "package-name (version1, version2, ...)"
	// Example: "bundler (2.4.10, 2.3.26)"
	gemRegex := regexp.MustCompile(`^(\S+)\s+\(([^)]+)\)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		matches := gemRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		packageName := matches[1]
		versions := strings.Split(matches[2], ", ")

		// If multiple versions are installed, create a package entry for each
		// (though typically we'll just report the first/latest)
		for _, version := range versions {
			version = strings.TrimSpace(version)

			packages = append(packages, &Package{
				Name:           packageName,
				Version:        version,
				Architecture:   "",            // gem is platform-independent
				PackageManager: "gem",
				Source:         "rubygems.org", // Default RubyGems registry
				InstalledAt:    time.Time{},   // Would require gem specification per package
				PackageSize:    0,              // Would require scanning gem directory
				Description:    "",             // Would require gem specification call
			})

			// Only report the first version (usually the latest)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse gem output: %w", err)
	}

	return packages, nil
}
