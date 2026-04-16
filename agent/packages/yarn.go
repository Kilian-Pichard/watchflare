package packages

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const yarnTimeout = 30 * time.Second

// yarnInfoRegex matches the package@version string from yarn info events.
// Yarn 1.x outputs: {"type":"info","data":"\"typescript@6.0.2\" has binaries:"}
var yarnInfoRegex = regexp.MustCompile(`^"([^"]+)" has binaries:`)

// YarnCollector collects globally installed Yarn packages
type YarnCollector struct {
	yarnPath string
}

// Name returns the collector name
func (y *YarnCollector) Name() string {
	return "yarn-global"
}

// IsAvailable checks if yarn is available
func (y *YarnCollector) IsAvailable() bool {
	path, err := exec.LookPath("yarn")
	if err != nil {
		return false
	}
	y.yarnPath = path
	return true
}

// Collect gathers all globally installed Yarn packages.
// Uses "yarn global list --json" (NDJSON output).
func (y *YarnCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), yarnTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, y.yarnPath, "global", "list", "--json")
	cmd.Env = yarnEnvWithDirs()
	cmd.Dir = "/tmp/watchflare-yarn"
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yarn global list failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal(line, &data); err != nil {
			continue
		}
		switch data["type"] {
		case "tree":
			// Older yarn 1.x format: one tree event with all packages.
			dataObj, _ := data["data"].(map[string]interface{})
			trees, _ := dataObj["trees"].([]interface{})
			for _, tree := range trees {
				if treeMap, ok := tree.(map[string]interface{}); ok {
					if pkg := parseYarnTreeNode(treeMap); pkg != nil {
						packages = append(packages, pkg)
					}
				}
			}
		case "info":
			// Yarn 1.22+ format: one info event per package.
			// data: "\"typescript@6.0.2\" has binaries:"
			infoStr, _ := data["data"].(string)
			if pkg := parseYarnInfoLine(infoStr); pkg != nil {
				packages = append(packages, pkg)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading yarn output: %w", err)
	}

	return packages, nil
}

// parseYarnTreeNode parses a Yarn tree node into a Package.
func parseYarnTreeNode(node map[string]interface{}) *Package {
	nameStr, _ := node["name"].(string)
	if nameStr == "" {
		return nil
	}
	name, version := parseYarnNameVersion(nameStr)
	return &Package{
		Name:           name,
		Version:        version,
		PackageManager: "yarn-global",
		Source:         "npmjs.com",
	}
}

// yarnEnvWithDirs returns the environment for yarn commands with HOME and
// YARN_CACHE_FOLDER redirected to /tmp. When the service user has a non-writable
// home (e.g. /var/empty), yarn fails trying to read ~/.yarnrc.
func yarnEnvWithDirs() []string {
	const tmpDir = "/tmp/watchflare-yarn"
	_ = os.MkdirAll(tmpDir, 0700)

	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "YARN_CACHE_FOLDER=") {
			continue
		}
		env = append(env, e)
	}
	return append(env,
		"HOME="+tmpDir,
		"YARN_CACHE_FOLDER="+tmpDir,
	)
}

// parseYarnInfoLine parses a yarn 1.22+ info line into a Package.
// Input: "\"typescript@6.0.2\" has binaries:"
func parseYarnInfoLine(infoStr string) *Package {
	matches := yarnInfoRegex.FindStringSubmatch(infoStr)
	if len(matches) < 2 {
		return nil
	}
	name, version := parseYarnNameVersion(matches[1])
	if name == "" || version == "" {
		return nil
	}
	return &Package{
		Name:           name,
		Version:        version,
		PackageManager: "yarn-global",
		Source:         "npmjs.com",
	}
}

// parseYarnNameVersion splits a "package@version" string into name and version.
// Handles scoped packages: "@scope/name@1.0.0" → ("@scope/name", "1.0.0").
// Uses the last "@" as the separator (idx > 0 excludes the leading "@" of scoped names).
func parseYarnNameVersion(nameVersion string) (string, string) {
	idx := strings.LastIndex(nameVersion, "@")
	if idx > 0 {
		return nameVersion[:idx], nameVersion[idx+1:]
	}
	return nameVersion, ""
}
