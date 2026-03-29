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

const nugetTimeout = 30 * time.Second

// NuGetCollector collects globally installed .NET tools
type NuGetCollector struct {
	dotnetPath string
}

// Name returns the collector name
func (n *NuGetCollector) Name() string {
	return "nuget-global"
}

// IsAvailable checks if dotnet is available
func (n *NuGetCollector) IsAvailable() bool {
	dotnetPath, err := exec.LookPath("dotnet")
	if err != nil {
		return false
	}
	n.dotnetPath = dotnetPath
	return true
}

// Collect gathers all globally installed .NET tools.
// Parses "dotnet tool list -g" output:
//
//	Package Id        Version    Commands
//	------------------------------------------
//	dotnet-ef         7.0.0      dotnet-ef
func (n *NuGetCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), nugetTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, n.dotnetPath, "tool", "list", "-g")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dotnet tool list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	headerPassed := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !headerPassed {
			if strings.Contains(line, "---") {
				headerPassed = true
			}
			continue
		}
		if pkg := parseNuGetLine(line); pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read dotnet output: %w", err)
	}

	return packages, nil
}

// parseNuGetLine parses a single data line of dotnet tool list output.
// Format: "package-id  version  commands"
func parseNuGetLine(line string) *Package {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	description := ""
	if len(fields) > 2 {
		description = strings.Join(fields[2:], ", ")
	}

	return &Package{
		Name:           fields[0],
		Version:        fields[1],
		PackageManager: "nuget-global",
		Source:         "nuget.org",
		Description:    description,
	}
}
