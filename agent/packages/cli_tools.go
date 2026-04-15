package packages

import (
	"context"
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
	{"mvn", []string{"--version"}, "build"},
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
	{"curl", []string{"--version"}, "network"},
	{"wget", []string{"--version"}, "network"},
	{"jq", []string{"--version"}, "monitoring"},
}

// versionTimeout is the maximum time allowed per version command invocation.
const versionTimeout = 5 * time.Second

// versionPatterns are compiled once at package init and reused across calls.
// Patterns are ordered from most specific (X.Y.Z) to least specific (X.Y).
var versionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)version\s+v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`),
	regexp.MustCompile(`(?i)^v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`),
	regexp.MustCompile(`(?i)^\w+\s+v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`),
	regexp.MustCompile(`^v?(\d+\.\d+\.\d+[a-zA-Z0-9._-]*)`),
	regexp.MustCompile(`v?(\d+\.\d+\.\d+)`),
	// X.Y fallback for tools like jq that output "jq-1.6"
	regexp.MustCompile(`(?i)version\s+v?(\d+\.\d+)`),
	regexp.MustCompile(`[^\d]v?(\d+\.\d+)\s*$`),
}

var semverFallback = regexp.MustCompile(`\d+\.\d+\.\d+`)

// Collect gathers versions of all available CLI tools
func (c *CLIToolsCollector) Collect() ([]*Package, error) {
	var packages []*Package

	for _, tool := range predefinedCLIs {
		toolPath, err := exec.LookPath(tool.name)
		if err != nil {
			continue
		}

		version := c.getVersion(toolPath, tool)
		if version == "" {
			continue
		}

		packages = append(packages, &Package{
			Name:           tool.name,
			Version:        version,
			PackageManager: "cli-tools",
			Source:         tool.category,
			Description:    toolPath,
		})
	}

	return packages, nil
}

// getVersion tries multiple version commands for the given binary path.
func (c *CLIToolsCollector) getVersion(toolPath string, tool cliTool) string {
	for _, versionCmd := range tool.versionCommands {
		version := c.tryVersionCommand(toolPath, versionCmd)
		if version != "" {
			return version
		}
	}
	return ""
}

// tryVersionCommand executes a version command with a timeout and parses output.
// toolPath must be the resolved binary path (from exec.LookPath).
func (c *CLIToolsCollector) tryVersionCommand(toolPath, versionCmd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), versionTimeout)
	defer cancel()

	args := strings.Fields(versionCmd)
	cmd := exec.CommandContext(ctx, toolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	return c.parseVersion(string(output))
}

// parseVersion extracts a semantic version number from command output
func (c *CLIToolsCollector) parseVersion(output string) string {
	for _, re := range versionPatterns {
		if matches := re.FindStringSubmatch(output); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Fallback: any x.y.z anywhere in output
	if match := semverFallback.FindString(output); match != "" {
		return match
	}

	return ""
}
