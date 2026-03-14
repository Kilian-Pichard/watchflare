package packages

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// PipxCollector collects globally installed pipx applications
// pipx is used to install Python CLI applications in isolated environments
type PipxCollector struct{}

// Name returns the collector name
func (p *PipxCollector) Name() string {
	return "pipx"
}

// IsAvailable checks if pipx is available
func (p *PipxCollector) IsAvailable() bool {
	_, err := exec.LookPath("pipx")
	return err == nil
}

// Collect gathers all installed pipx applications
func (p *PipxCollector) Collect() ([]*Package, error) {
	// Run pipx list with JSON output
	cmd := exec.Command("pipx", "list", "--json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pipx list failed: %w", err)
	}

	// Parse JSON output
	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("failed to parse pipx output: %w", err)
	}

	var packages []*Package

	// pipx JSON structure: {"venvs": {"app-name": {...}, ...}}
	if venvs, ok := data["venvs"].(map[string]interface{}); ok {
		for appName, venvData := range venvs {
			if venvMap, ok := venvData.(map[string]interface{}); ok {
				pkg := p.parseVenvData(appName, venvMap)
				if pkg != nil {
					packages = append(packages, pkg)
				}
			}
		}
	}

	return packages, nil
}

// parseVenvData parses a pipx venv JSON object into a Package
func (p *PipxCollector) parseVenvData(appName string, venvData map[string]interface{}) *Package {
	// Extract metadata
	metadata, hasMetadata := venvData["metadata"].(map[string]interface{})

	version := ""
	pythonVersion := ""

	if hasMetadata {
		// Get main package info
		if mainPkg, ok := metadata["main_package"].(map[string]interface{}); ok {
			if pkgVersion, ok := mainPkg["package_version"].(string); ok {
				version = pkgVersion
			}
			if appName == "" {
				if pkgName, ok := mainPkg["package"].(string); ok {
					appName = pkgName
				}
			}
		}

		// Get Python version
		if pyVer, ok := metadata["python_version"].(string); ok {
			pythonVersion = pyVer
		}
	}

	// Get installed apps/commands
	commands := []string{}
	if apps, ok := venvData["apps"].([]interface{}); ok {
		for _, app := range apps {
			if appStr, ok := app.(string); ok {
				commands = append(commands, appStr)
			}
		}
	}

	description := fmt.Sprintf("Python CLI app (Python %s)", pythonVersion)
	if len(commands) > 0 {
		description = fmt.Sprintf("Commands: %v (Python %s)", commands, pythonVersion)
	}

	return &Package{
		Name:           appName,
		Version:        version,
		Architecture:   "",
		PackageManager: "pipx",
		Source:         "pypi", // pipx installs from PyPI
		InstalledAt:    time.Time{},
		PackageSize:    0,
		Description:    description,
	}
}
