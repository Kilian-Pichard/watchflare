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

const mambaTimeout = 30 * time.Second

// MambaCollector collects packages from the mamba base environment.
// Mamba is a fast drop-in replacement for conda with identical output format.
type MambaCollector struct {
	mambaPath string
}

// Name returns the collector name
func (m *MambaCollector) Name() string {
	return "mamba"
}

// IsAvailable checks if mamba is available
func (m *MambaCollector) IsAvailable() bool {
	mambaPath, err := exec.LookPath("mamba")
	if err != nil {
		return false
	}
	m.mambaPath = mambaPath
	return true
}

// Collect gathers all packages from the mamba base environment.
// Parses "mamba list" output (identical format to conda list):
//
//	numpy   1.24.3   py311h08b1b3b_0   conda-forge
func (m *MambaCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mambaTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.mambaPath, "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("mamba list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if pkg := parseMambaLine(line); pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read mamba output: %w", err)
	}

	return packages, nil
}

// parseMambaLine parses a single line of mamba list output.
func parseMambaLine(line string) *Package { return parseCondaStyleLine(line, "mamba") }
