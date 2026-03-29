package packages

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const yarnTimeout = 30 * time.Second

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
		if data["type"] != "tree" {
			continue
		}
		dataObj, _ := data["data"].(map[string]interface{})
		trees, _ := dataObj["trees"].([]interface{})
		for _, tree := range trees {
			if treeMap, ok := tree.(map[string]interface{}); ok {
				if pkg := parseYarnTreeNode(treeMap); pkg != nil {
					packages = append(packages, pkg)
				}
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
