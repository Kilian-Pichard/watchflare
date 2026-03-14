package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// MambaCollector collects packages from mamba environments
// mamba is a fast drop-in replacement for conda
type MambaCollector struct{}

// Name returns the collector name
func (m *MambaCollector) Name() string {
	return "mamba"
}

// IsAvailable checks if mamba is available
func (m *MambaCollector) IsAvailable() bool {
	_, err := exec.LookPath("mamba")
	return err == nil
}

// Collect gathers all packages from mamba base environment
func (m *MambaCollector) Collect() ([]*Package, error) {
	// Run mamba list to get installed packages
	cmd := exec.Command("mamba", "list")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("mamba list failed: %w", err)
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
		pkg := m.parsePackageLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading mamba output: %w", err)
	}

	return packages, nil
}

// parsePackageLine parses a single line of mamba list output
func (m *MambaCollector) parsePackageLine(line string) *Package {
	// Format: "name  version  build  channel"
	// Example: "numpy  1.24.3  py311h08b1b3b_0  conda-forge"
	// Same format as conda

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

	// Extract size if available
	var size int64
	for i := 4; i < len(fields); i++ {
		if strings.Contains(fields[i], "KB") || strings.Contains(fields[i], "MB") {
			size = m.parseSize(fields[i])
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
		PackageManager: "mamba",
		Source:         channel,
		InstalledAt:    time.Time{},
		PackageSize:    size,
		Description:    description,
	}
}

// parseSize converts size strings like "1.2MB" or "345KB" to bytes
func (m *MambaCollector) parseSize(sizeStr string) int64 {
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
