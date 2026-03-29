package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	poetryConfigTimeout = 10 * time.Second
	poetryPipTimeout    = 30 * time.Second
)

// PoetryCollector collects packages from Poetry virtual environments
type PoetryCollector struct {
	poetryPath string
}

// Name returns the collector name
func (p *PoetryCollector) Name() string {
	return "poetry"
}

// IsAvailable checks if poetry is available
func (p *PoetryCollector) IsAvailable() bool {
	path, err := exec.LookPath("poetry")
	if err != nil {
		return false
	}
	p.poetryPath = path
	return true
}

// Collect gathers packages from all Poetry virtual environments.
func (p *PoetryCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), poetryConfigTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.poetryPath, "config", "cache-dir")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get poetry cache dir: %w", err)
	}

	virtualenvsDir := filepath.Join(strings.TrimSpace(string(output)), "virtualenvs")
	if _, err := os.Stat(virtualenvsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(virtualenvsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read virtualenvs dir: %w", err)
	}

	var packages []*Package
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		envName := entry.Name()
		envPath := filepath.Join(virtualenvsDir, envName)
		envPackages, err := listVirtualenvPackages(envPath, envName)
		if err != nil {
			continue
		}
		packages = append(packages, envPackages...)
	}

	return packages, nil
}

// listVirtualenvPackages lists packages in a Poetry virtualenv using pip freeze.
func listVirtualenvPackages(envPath, envName string) ([]*Package, error) {
	pipPath := filepath.Join(envPath, "bin", "pip")
	if _, err := os.Stat(pipPath); os.IsNotExist(err) {
		pipPath = filepath.Join(envPath, "Scripts", "pip.exe")
		if _, err := os.Stat(pipPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("pip not found in virtualenv")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), poetryPipTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pipPath, "list", "--format=freeze")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		name, version, ok := parsePipFreezeLine(scanner.Text())
		if !ok {
			continue
		}
		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			PackageManager: "poetry",
			Source:         envName,
			Description:    fmt.Sprintf("Poetry virtualenv: %s", envName),
		})
	}

	return packages, nil
}

// parsePipFreezeLine parses a "pip list --format=freeze" line: "package==version".
func parsePipFreezeLine(line string) (name, version string, ok bool) {
	if line == "" {
		return
	}
	parts := strings.SplitN(line, "==", 2)
	if len(parts) != 2 {
		return
	}
	return parts[0], parts[1], true
}
