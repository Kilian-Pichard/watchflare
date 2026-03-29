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

const condaTimeout = 30 * time.Second

// CondaCollector collects packages from the conda base environment
// (Anaconda/Miniconda, data science focused)
type CondaCollector struct {
	condaPath string
}

// Name returns the collector name
func (c *CondaCollector) Name() string {
	return "conda"
}

// IsAvailable checks if conda is available
func (c *CondaCollector) IsAvailable() bool {
	condaPath, err := exec.LookPath("conda")
	if err != nil {
		return false
	}
	c.condaPath = condaPath
	return true
}

// Collect gathers all packages from the conda base environment.
// Parses "conda list" output:
//
//	numpy   1.24.3   py311h08b1b3b_0   conda-forge
func (c *CondaCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), condaTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.condaPath, "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("conda list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if pkg := parseCondaLine(line); pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read conda output: %w", err)
	}

	return packages, nil
}

// parseCondaStyleLine parses a single line of conda/mamba list output.
// Format: "name  version  build  channel"
// pm is the PackageManager value to use ("conda" or "mamba").
func parseCondaStyleLine(line, pm string) *Package {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	build := ""
	channel := ""
	if len(fields) >= 3 {
		build = fields[2]
	}
	if len(fields) >= 4 {
		channel = fields[3]
	}

	description := fmt.Sprintf("Build: %s", build)
	if channel != "" {
		description = fmt.Sprintf("Channel: %s, Build: %s", channel, build)
	}

	return &Package{
		Name:           fields[0],
		Version:        fields[1],
		PackageManager: pm,
		Source:         channel,
		Description:    description,
	}
}

// parseCondaLine parses a single line of conda list output.
func parseCondaLine(line string) *Package { return parseCondaStyleLine(line, "conda") }
