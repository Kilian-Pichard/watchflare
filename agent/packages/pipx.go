package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const pipxTimeout = 30 * time.Second

// PipxCollector collects globally installed pipx applications.
// pipx installs Python CLI applications in isolated virtual environments.
type PipxCollector struct {
	pipxPath string
}

// Name returns the collector name
func (p *PipxCollector) Name() string {
	return "pipx"
}

// IsAvailable checks if pipx is available
func (p *PipxCollector) IsAvailable() bool {
	path, err := exec.LookPath("pipx")
	if err != nil {
		return false
	}
	p.pipxPath = path
	return true
}

// Collect gathers all installed pipx applications.
// Uses "pipx list --json".
func (p *PipxCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pipxTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.pipxPath, "list", "--json")
	cmd.Env = pipxEnvWithDirs()
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pipx list failed: %w", err)
	}

	return parsePipxOutput(output)
}

// pipxEnvWithDirs returns the environment for pipx commands with HOME and PIPX_HOME
// redirected to /tmp. When the service user has a non-writable home (e.g. /var/empty),
// pipx fails trying to access ~/.local/share/pipx. Redirecting both HOME and PIPX_HOME
// to a writable temp directory prevents this.
func pipxEnvWithDirs() []string {
	const tmpDir = "/tmp/watchflare-pipx"
	_ = os.MkdirAll(tmpDir, 0700)

	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "PIPX_HOME=") {
			continue
		}
		env = append(env, e)
	}
	return append(env,
		"HOME="+tmpDir,
		"PIPX_HOME="+tmpDir,
	)
}

// parsePipxOutput parses the JSON output of "pipx list --json".
// Structure: {"venvs": {"app-name": {...}, ...}}
func parsePipxOutput(output []byte) ([]*Package, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("failed to parse pipx output: %w", err)
	}

	venvs, _ := data["venvs"].(map[string]interface{})
	packages := make([]*Package, 0, len(venvs))
	for appName, venvData := range venvs {
		if venvMap, ok := venvData.(map[string]interface{}); ok {
			if pkg := parsePipxVenv(appName, venvMap); pkg != nil {
				packages = append(packages, pkg)
			}
		}
	}

	return packages, nil
}

// parsePipxVenv parses a single pipx venv JSON block into a Package.
func parsePipxVenv(appName string, venvData map[string]interface{}) *Package {
	version := ""
	pythonVersion := ""

	var commands []string
	if metadata, ok := venvData["metadata"].(map[string]interface{}); ok {
		if mainPkg, ok := metadata["main_package"].(map[string]interface{}); ok {
			version, _ = mainPkg["package_version"].(string)
			if appName == "" {
				appName, _ = mainPkg["package"].(string)
			}
			if apps, ok := mainPkg["apps"].([]interface{}); ok {
				for _, app := range apps {
					if appStr, ok := app.(string); ok {
						commands = append(commands, appStr)
					}
				}
			}
		}
		pythonVersion, _ = metadata["python_version"].(string)
	}

	var description string
	if len(commands) > 0 {
		description = fmt.Sprintf("Commands: %s (Python %s)", strings.Join(commands, ", "), pythonVersion)
	} else {
		description = fmt.Sprintf("Python CLI app (Python %s)", pythonVersion)
	}

	return &Package{
		Name:           appName,
		Version:        version,
		PackageManager: "pipx",
		Source:         "pypi.org",
		Description:    description,
	}
}
