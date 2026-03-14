package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CondaCollector collects packages from conda environments
// conda is the package manager for Anaconda/Miniconda (data science focused)
type CondaCollector struct{}

// Name returns the collector name
func (c *CondaCollector) Name() string {
	return "conda"
}

// IsAvailable checks if conda is available
func (c *CondaCollector) IsAvailable() bool {
	_, err := exec.LookPath("conda")
	return err == nil
}

// Collect gathers all packages from conda base environment
func (c *CondaCollector) Collect() ([]*Package, error) {
	// Get list of packages in base environment
	// Using --json for easier parsing
	cmd := exec.Command("conda", "list", "--json")

	output, err := cmd.Output()
	if err != nil {
		// If JSON fails, try regular format
		return c.collectRegularFormat()
	}

	// Try to parse as JSON first
	packages, err := c.parseJSON(output)
	if err != nil {
		// Fallback to regular format
		return c.collectRegularFormat()
	}

	return packages, nil
}

// parseJSON parses conda list JSON output
func (c *CondaCollector) parseJSON(output []byte) ([]*Package, error) {
	// conda list --json returns an array of objects
	// [{"name": "pkg", "version": "1.0", "build": "...", "channel": "..."}]

	// Simple JSON array parsing
	jsonStr := string(output)
	jsonStr = strings.TrimSpace(jsonStr)

	if !strings.HasPrefix(jsonStr, "[") {
		return nil, fmt.Errorf("not a JSON array")
	}

	// For simplicity, we'll parse manually
	// This is a simplified approach - for production, use encoding/json
	return c.collectRegularFormat()
}

// collectRegularFormat parses conda list regular output
func (c *CondaCollector) collectRegularFormat() ([]*Package, error) {
	cmd := exec.Command("conda", "list")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("conda list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header lines (starting with #)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Parse package line
		pkg := c.parsePackageLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading conda output: %w", err)
	}

	return packages, nil
}

// parsePackageLine parses a single line of conda list output
func (c *CondaCollector) parsePackageLine(line string) *Package {
	// Format: "name  version  build  channel"
	// Example: "numpy  1.24.3  py311h08b1b3b_0  conda-forge"

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	name := fields[0]
	version := fields[1]
	build := ""
	channel := ""

	if len(fields) >= 3 {
		build = fields[2]
	}
	if len(fields) >= 4 {
		channel = fields[3]
	}

	// Extract size if available (some formats include it)
	var size int64
	for i := 4; i < len(fields); i++ {
		if strings.Contains(fields[i], "KB") || strings.Contains(fields[i], "MB") {
			size = c.parseSize(fields[i])
		}
	}

	description := fmt.Sprintf("Build: %s", build)
	if channel != "" {
		description = fmt.Sprintf("Channel: %s, Build: %s", channel, build)
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",
		PackageManager: "conda",
		Source:         channel,
		InstalledAt:    time.Time{},
		PackageSize:    size,
		Description:    description,
	}
}

// parseSize converts size strings like "1.2MB" or "345KB" to bytes
func (c *CondaCollector) parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	sizeStr = strings.ToUpper(sizeStr)

	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(sizeStr), 64)
	if err != nil {
		return 0
	}

	return int64(value * float64(multiplier))
}
