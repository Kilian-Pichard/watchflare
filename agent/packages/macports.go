package packages

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MacPortsCollector collects packages from MacPorts (macOS)
type MacPortsCollector struct {
	portPath string
}

// Name returns the collector name
func (m *MacPortsCollector) Name() string {
	return "macports"
}

// IsAvailable checks if port is available
func (m *MacPortsCollector) IsAvailable() bool {
	// MacPorts installs to /opt/local by default
	portPath := "/opt/local/bin/port"
	if _, err := os.Stat(portPath); err == nil {
		m.portPath = portPath
		return true
	}

	return false
}

// Collect gathers all installed packages from MacPorts
func (m *MacPortsCollector) Collect() ([]*Package, error) {
	// Run port installed to get list of installed ports
	cmd := exec.Command(m.portPath, "installed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("port installed failed: %w (output: %s)", err, string(output))
	}

	var pkgs []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header line ("The following ports are currently installed:")
	if scanner.Scan() {
		// Skip the header
	}

	// Pattern: "  package-name @version+variants (active)" or "  package-name @version"
	// Example: "  git @2.43.0_0+credential_osxkeychain+diff_highlight+pcre+perl5_34 (active)"
	portRegex := regexp.MustCompile(`^\s+(\S+)\s+@([^\s\+]+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		matches := portRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		packageName := matches[1]
		version := matches[2]

		// Get package details
		pkg, err := m.getPackageInfo(packageName, version)
		if err != nil {
			// Skip packages that fail to query
			continue
		}

		pkgs = append(pkgs, pkg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse port output: %w", err)
	}

	return pkgs, nil
}

// getPackageInfo retrieves detailed information for a single package
func (m *MacPortsCollector) getPackageInfo(name, version string) (*Package, error) {
	// Get port info
	cmd := exec.Command(m.portPath, "info", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("port info failed for %s: %w", name, err)
	}

	// Parse port info output
	// Example output:
	// git @2.43.0 (devel)
	// Variants:             credential_osxkeychain, diff_highlight, pcre, ...
	//
	// Description:          Git is a fast, scalable, distributed revision control system...
	// Homepage:             https://git-scm.com/
	//
	// Library Dependencies: pcre2, ...
	// Platforms:            darwin
	// License:              GPL-2
	// Maintainers:          Email: dports@macports.org

	var description string
	var size int64

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	inDescription := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Description:") {
			inDescription = true
			description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
			continue
		}

		if inDescription {
			if strings.TrimSpace(line) == "" || strings.Contains(line, ":") {
				inDescription = false
			} else {
				description += " " + strings.TrimSpace(line)
			}
		}
	}

	// Try to get installed size from port contents
	if installedSize := m.getInstalledSize(name); installedSize > 0 {
		size = installedSize
	}

	// Truncate description to 100 chars
	description = TruncateDescription(description)

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   "",        // MacPorts doesn't provide arch info easily
		PackageManager: "macports",
		Source:         "macports", // Default source
		InstalledAt:    time.Time{}, // MacPorts doesn't easily provide install date
		PackageSize:    size,
		Description:    description,
	}, nil
}

// getInstalledSize estimates installed size by counting files
func (m *MacPortsCollector) getInstalledSize(name string) int64 {
	// Run port contents to get list of installed files
	cmd := exec.Command(m.portPath, "contents", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	var totalSize int64
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "/") {
			continue
		}

		// Get file size
		info, err := os.Stat(line)
		if err != nil {
			continue
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}
	}

	return totalSize
}

// parseInt64 safely parses a string to int64, returning 0 on error
func parseInt64MacPorts(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}

	return val
}
