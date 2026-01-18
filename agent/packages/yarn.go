package packages

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// YarnCollector collects globally installed Yarn packages
type YarnCollector struct{}

// Name returns the collector name
func (y *YarnCollector) Name() string {
	return "yarn-global"
}

// IsAvailable checks if yarn is available
func (y *YarnCollector) IsAvailable() bool {
	_, err := exec.LookPath("yarn")
	return err == nil
}

// Collect gathers all globally installed Yarn packages
func (y *YarnCollector) Collect() ([]*Package, error) {
	// Run yarn global list with JSON output
	cmd := exec.Command("yarn", "global", "list", "--json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yarn global list failed: %w", err)
	}

	// Yarn outputs NDJSON (newline-delimited JSON)
	// We need to parse each line as a separate JSON object
	var packages []*Package
	lines := splitLines(string(output))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		// Look for "data" entries with package info
		if data["type"] == "tree" {
			if dataObj, ok := data["data"].(map[string]interface{}); ok {
				if trees, ok := dataObj["trees"].([]interface{}); ok {
					for _, tree := range trees {
						if treeMap, ok := tree.(map[string]interface{}); ok {
							pkg := y.parseTreeNode(treeMap)
							if pkg != nil {
								packages = append(packages, pkg)
							}
						}
					}
				}
			}
		}
	}

	return packages, nil
}

// parseTreeNode parses a Yarn tree node into a Package
func (y *YarnCollector) parseTreeNode(node map[string]interface{}) *Package {
	nameStr, hasName := node["name"].(string)
	if !hasName || nameStr == "" {
		return nil
	}

	// Yarn format: "package@version"
	name, version := parseNameVersion(nameStr)

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",
		PackageManager: "yarn-global",
		Source:         "yarn",
		InstalledAt:    time.Time{},
		PackageSize:    0,
		Description:    "Globally installed Yarn package",
	}
}

// parseNameVersion splits "package@version" into name and version
func parseNameVersion(nameVersion string) (string, string) {
	// Handle scoped packages: "@scope/package@version"
	if nameVersion[0] == '@' {
		// Find the second @ which separates package from version
		firstSlash := -1
		for i, c := range nameVersion {
			if c == '/' {
				firstSlash = i
				break
			}
		}
		if firstSlash == -1 {
			return nameVersion, ""
		}

		// Find @ after the slash
		for i := firstSlash; i < len(nameVersion); i++ {
			if nameVersion[i] == '@' {
				return nameVersion[:i], nameVersion[i+1:]
			}
		}
		return nameVersion, ""
	}

	// Regular package: "package@version"
	for i := len(nameVersion) - 1; i >= 0; i-- {
		if nameVersion[i] == '@' {
			return nameVersion[:i], nameVersion[i+1:]
		}
	}

	return nameVersion, ""
}
