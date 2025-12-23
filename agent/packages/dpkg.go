package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DpkgCollector collects packages from dpkg (Debian/Ubuntu)
type DpkgCollector struct{}

// Name returns the collector name
func (d *DpkgCollector) Name() string {
	return "dpkg"
}

// IsAvailable checks if dpkg-query is available
func (d *DpkgCollector) IsAvailable() bool {
	_, err := exec.LookPath("dpkg-query")
	return err == nil
}

// Collect gathers all installed packages from dpkg
func (d *DpkgCollector) Collect() ([]*Package, error) {
	// Run dpkg-query to get package information
	// Format: Package|Version|Architecture|Installed-Size|Status|Description
	cmd := exec.Command("dpkg-query", "-W",
		"-f=${Package}|${Version}|${Architecture}|${Installed-Size}|${db:Status-Abbrev}|${Description}\n")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dpkg-query failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 6 {
			continue
		}

		name := fields[0]
		version := fields[1]
		arch := fields[2]
		sizeStr := fields[3]
		status := strings.TrimSpace(fields[4])
		description := fields[5]

		// Only include installed packages (status: "ii " = installed ok installed)
		if !strings.HasPrefix(status, "ii") {
			continue
		}

		// Parse size (dpkg reports in KB, convert to bytes)
		size := parseInt64(sizeStr) * 1024

		// Truncate description to 100 chars
		description = TruncateDescription(description)

		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			Architecture:   arch,
			PackageManager: "dpkg",
			Source:         "", // dpkg doesn't easily provide repo info
			InstalledAt:    time.Time{}, // dpkg doesn't track installation date
			PackageSize:    size,
			Description:    description,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse dpkg output: %w", err)
	}

	return packages, nil
}

// parseInt64 safely parses a string to int64, returning 0 on error
func parseInt64(s string) int64 {
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
