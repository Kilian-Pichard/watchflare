package packages

import (
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// CLIToolsCollector collects versions of important CLI tools
type CLIToolsCollector struct{}

// Name returns the collector name
func (c *CLIToolsCollector) Name() string {
	return "cli-tools"
}

// IsAvailable is always true (cross-platform)
func (c *CLIToolsCollector) IsAvailable() bool {
	return true
}

// cliTool represents a CLI tool to track
type cliTool struct {
	name            string   // Binary name (e.g., "docker")
	versionCommands []string // Commands to try for version (e.g., ["--version", "-v"])
	category        string   // Category for organization
}

// predefinedCLIs is the list of important CLI tools to track
var predefinedCLIs = []cliTool{
	// Containers & Orchestration
	{"docker", []string{"--version", "-v", "version --short"}, "containers"},
	{"docker-compose", []string{"--version", "-v"}, "containers"},
	{"podman", []string{"--version", "-v"}, "containers"},
	{"kubectl", []string{"version --client --short", "version --client"}, "containers"},
	{"helm", []string{"version --short", "version"}, "containers"},
	{"minikube", []string{"version --short", "version"}, "containers"},
	{"k3s", []string{"--version"}, "containers"},
	{"kind", []string{"version"}, "containers"},

	// Cloud CLIs
	{"aws", []string{"--version"}, "cloud"},
	{"gcloud", []string{"version --format=json"}, "cloud"},
	{"az", []string{"version"}, "cloud"},
	{"terraform", []string{"version"}, "cloud"},
	{"pulumi", []string{"version"}, "cloud"},
	{"doctl", []string{"version"}, "cloud"},
	{"ibmcloud", []string{"version"}, "cloud"},

	// Languages & Runtimes
	{"node", []string{"--version", "-v"}, "languages"},
	{"python", []string{"--version"}, "languages"},
	{"python3", []string{"--version"}, "languages"},
	{"go", []string{"version"}, "languages"},
	{"rustc", []string{"--version"}, "languages"},
	{"cargo", []string{"--version"}, "languages"},
	{"java", []string{"--version", "-version"}, "languages"},
	{"javac", []string{"--version"}, "languages"},
	{"ruby", []string{"--version"}, "languages"},
	{"php", []string{"--version"}, "languages"},
	{"perl", []string{"--version"}, "languages"},
	{"dotnet", []string{"--version"}, "languages"},

	// Version Control
	{"git", []string{"--version"}, "vcs"},
	{"gh", []string{"--version"}, "vcs"},
	{"gitlab", []string{"--version"}, "vcs"},
	{"svn", []string{"--version"}, "vcs"},
	{"hg", []string{"--version"}, "vcs"},

	// Build Tools
	{"make", []string{"--version"}, "build"},
	{"cmake", []string{"--version"}, "build"},
	{"gradle", []string{"--version"}, "build"},
	{"maven", []string{"--version"}, "build"},
	{"ant", []string{"--version"}, "build"},
	{"ninja", []string{"--version"}, "build"},

	// Compilers
	{"gcc", []string{"--version"}, "compilers"},
	{"g++", []string{"--version"}, "compilers"},
	{"clang", []string{"--version"}, "compilers"},
	{"clang++", []string{"--version"}, "compilers"},

	// Databases
	{"mysql", []string{"--version"}, "databases"},
	{"psql", []string{"--version"}, "databases"},
	{"redis-cli", []string{"--version"}, "databases"},
	{"mongo", []string{"--version"}, "databases"},
	{"mongosh", []string{"--version"}, "databases"},
	{"sqlite3", []string{"--version"}, "databases"},

	// DevOps Tools
	{"ansible", []string{"--version"}, "devops"},
	{"vagrant", []string{"--version"}, "devops"},
	{"packer", []string{"--version"}, "devops"},
	{"consul", []string{"version"}, "devops"},
	{"vault", []string{"version"}, "devops"},
	{"nomad", []string{"version"}, "devops"},

	// Monitoring & Observability
	{"prometheus", []string{"--version"}, "monitoring"},
	{"grafana-cli", []string{"--version"}, "monitoring"},
	{"curl", []string{"--version"}, "monitoring"},
	{"wget", []string{"--version"}, "monitoring"},
	{"jq", []string{"--version"}, "monitoring"},
}

// Collect gathers versions of all available CLI tools
func (c *CLIToolsCollector) Collect() ([]*Package, error) {
	var packages []*Package

	for _, tool := range predefinedCLIs {
		// Check if tool exists in PATH
		toolPath, err := exec.LookPath(tool.name)
		if err != nil {
			// Tool not installed, skip
			continue
		}

		// Try to get version
		version := c.getVersion(tool)
		if version == "" {
			// Could not determine version, skip
			continue
		}

		packages = append(packages, &Package{
			Name:           tool.name,
			Version:        version,
			Architecture:   "",          // CLI tools are usually architecture-independent
			PackageManager: "cli-tools",
			Source:         tool.category, // Use category as source
			InstalledAt:    time.Time{},   // Cannot determine install date easily
			PackageSize:    0,              // Could stat the binary for size
			Description:    toolPath,       // Store path as description
		})
	}

	return packages, nil
}

// getVersion tries multiple commands to extract version
func (c *CLIToolsCollector) getVersion(tool cliTool) string {
	for _, versionCmd := range tool.versionCommands {
		version := c.tryVersionCommand(tool.name, versionCmd)
		if version != "" {
			return version
		}
	}
	return ""
}

// tryVersionCommand executes a version command and parses output
func (c *CLIToolsCollector) tryVersionCommand(toolName, versionCmd string) string {
	// Split command into parts
	parts := strings.Fields(versionCmd)
	args := []string{}
	if len(parts) > 0 {
		args = parts
	}

	// Execute command with timeout
	cmd := exec.Command(toolName, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	// Parse version from output
	return c.parseVersion(string(output))
}

// parseVersion extracts version number from command output
func (c *CLIToolsCollector) parseVersion(output string) string {
	// Common version patterns
	// [a-zA-Z0-9._-]* allows alphanumeric, dots, underscores, hyphens (but not commas, spaces, etc.)
	patterns := []string{
		// "version 1.2.3" or "v1.2.3"
		`(?i)version\s+v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`,
		`(?i)^v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`,

		// "Tool 1.2.3"
		`(?i)^\w+\s+v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`,

		// Just version numbers at start of line
		`^v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`,

		// Version anywhere in first line
		`v?(\d+\.\d+\.\d+)`,
	}

	// Try each pattern
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Fallback: look for semantic version anywhere
	re := regexp.MustCompile(`\d+\.\d+\.\d+`)
	if match := re.FindString(output); match != "" {
		return match
	}

	return ""
}
