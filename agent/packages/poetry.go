package packages

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PoetryCollector collects globally installed Poetry environments and packages
type PoetryCollector struct{}

// Name returns the collector name
func (p *PoetryCollector) Name() string {
	return "poetry"
}

// IsAvailable checks if poetry is available
func (p *PoetryCollector) IsAvailable() bool {
	_, err := exec.LookPath("poetry")
	return err == nil
}

// Collect gathers Poetry virtual environments and their packages
func (p *PoetryCollector) Collect() ([]*Package, error) {
	var packages []*Package

	// Poetry stores virtual envs in a cache directory
	// Get Poetry cache dir
	cmd := exec.Command("poetry", "config", "cache-dir")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get poetry cache dir: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	virtualenvsDir := filepath.Join(cacheDir, "virtualenvs")

	// Check if virtualenvs directory exists
	if _, err := os.Stat(virtualenvsDir); os.IsNotExist(err) {
		return packages, nil // No virtualenvs, return empty list
	}

	// List all virtualenvs
	entries, err := os.ReadDir(virtualenvsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read virtualenvs dir: %w", err)
	}

	// For each virtualenv, list packages
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		envName := entry.Name()
		envPath := filepath.Join(virtualenvsDir, envName)

		// Try to list packages in this virtualenv
		envPackages, err := p.listVirtualenvPackages(envPath, envName)
		if err != nil {
			continue // Skip this env on error
		}

		packages = append(packages, envPackages...)
	}

	return packages, nil
}

// listVirtualenvPackages lists packages in a Poetry virtualenv
func (p *PoetryCollector) listVirtualenvPackages(envPath, envName string) ([]*Package, error) {
	// Find the pip in this virtualenv
	// Poetry virtualenvs have structure: env-name-py3.x/bin/pip (Linux/macOS)
	pipPath := filepath.Join(envPath, "bin", "pip")
	if _, err := os.Stat(pipPath); os.IsNotExist(err) {
		// Try Windows path
		pipPath = filepath.Join(envPath, "Scripts", "pip.exe")
		if _, err := os.Stat(pipPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("pip not found in virtualenv")
		}
	}

	// Run pip list
	cmd := exec.Command(pipPath, "list", "--format=freeze")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse pip freeze format: package==version
		parts := strings.Split(line, "==")
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		version := parts[1]

		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			Architecture:   "",
			PackageManager: "poetry",
			Source:         envName, // Use virtualenv name as source
			InstalledAt:    time.Time{},
			PackageSize:    0,
			Description:    fmt.Sprintf("Poetry virtualenv: %s", envName),
		})
	}

	return packages, nil
}
