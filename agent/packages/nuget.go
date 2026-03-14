package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NuGetCollector collects globally installed .NET tools (NuGet packages)
type NuGetCollector struct{}

// Name returns the collector name
func (n *NuGetCollector) Name() string {
	return "nuget-global"
}

// IsAvailable checks if dotnet is available
func (n *NuGetCollector) IsAvailable() bool {
	_, err := exec.LookPath("dotnet")
	return err == nil
}

// Collect gathers all globally installed .NET tools
func (n *NuGetCollector) Collect() ([]*Package, error) {
	// Run dotnet tool list -g to get global tools
	cmd := exec.Command("dotnet", "tool", "list", "-g")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dotnet tool list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header lines
	headerPassed := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Skip header lines until we find the separator (---)
		if !headerPassed {
			if strings.Contains(line, "---") {
				headerPassed = true
			}
			continue
		}

		// Parse package line
		pkg := n.parsePackageLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading dotnet output: %w", err)
	}

	return packages, nil
}

// parsePackageLine parses a single line of dotnet tool list output
func (n *NuGetCollector) parsePackageLine(line string) *Package {
	// Format: "Package Id      Version      Commands"
	// Example: "dotnet-ef      7.0.0        dotnet-ef"

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	name := fields[0]
	version := fields[1]

	// Commands are the rest of the fields (optional)
	commands := ""
	if len(fields) > 2 {
		commands = strings.Join(fields[2:], ", ")
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",
		PackageManager: "nuget-global",
		Source:         "nuget.org", // Default NuGet source
		InstalledAt:    time.Time{},
		PackageSize:    0,
		Description:    fmt.Sprintf("Commands: %s", commands),
	}
}
