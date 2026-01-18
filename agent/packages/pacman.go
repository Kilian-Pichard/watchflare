package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PacmanCollector collects packages from pacman (Arch Linux)
type PacmanCollector struct{}

// Name returns the collector name
func (p *PacmanCollector) Name() string {
	return "pacman"
}

// IsAvailable checks if pacman is available
func (p *PacmanCollector) IsAvailable() bool {
	_, err := exec.LookPath("pacman")
	return err == nil
}

// Collect gathers all installed packages from pacman
func (p *PacmanCollector) Collect() ([]*Package, error) {
	// Run pacman -Q to get all installed packages
	cmd := exec.Command("pacman", "-Q")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pacman -Q failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Format: "package-name version"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		version := fields[1]

		// Get detailed info for this package
		info := p.getPackageInfo(name)

		installDate := time.Time{}
		if info.InstallDate != nil {
			installDate = *info.InstallDate
		}

		packages = append(packages, &Package{
			Name:           name,
			Version:        version,
			Architecture:   info.Architecture,
			PackageManager: "pacman",
			Source:         info.Repository,
			InstalledAt:    installDate,
			PackageSize:    info.InstalledSize,
			Description:    info.Description,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading pacman output: %w", err)
	}

	return packages, nil
}

// packageInfo holds detailed package information
type packageInfo struct {
	Architecture  string
	Repository    string
	InstallDate   *time.Time
	InstalledSize int64
	Description   string
}

// getPackageInfo retrieves detailed information for a package
func (p *PacmanCollector) getPackageInfo(name string) packageInfo {
	info := packageInfo{}

	// Run pacman -Qi to get detailed info
	cmd := exec.Command("pacman", "-Qi", name)
	output, err := cmd.Output()
	if err != nil {
		return info
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Architecture") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Architecture = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Repository") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Repository = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Install Date") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dateStr := strings.TrimSpace(parts[1])
				// Try to parse the date (format: "Mon 02 Jan 2006 03:04:05 PM MST")
				if t, err := time.Parse("Mon 02 Jan 2006 03:04:05 PM MST", dateStr); err == nil {
					info.InstallDate = &t
				}
			}
		} else if strings.HasPrefix(line, "Installed Size") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				sizeStr := strings.TrimSpace(parts[1])
				// Parse size (e.g., "1.23 MiB" or "456.78 KiB")
				info.InstalledSize = parsePacmanSize(sizeStr)
			}
		} else if strings.HasPrefix(line, "Description") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Description = strings.TrimSpace(parts[1])
			}
		}
	}

	return info
}

// parsePacmanSize converts pacman size strings to bytes
func parsePacmanSize(sizeStr string) int64 {
	parts := strings.Fields(sizeStr)
	if len(parts) != 2 {
		return 0
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToLower(parts[1])
	multiplier := int64(1)

	switch unit {
	case "kib":
		multiplier = 1024
	case "mib":
		multiplier = 1024 * 1024
	case "gib":
		multiplier = 1024 * 1024 * 1024
	case "b":
		multiplier = 1
	}

	return int64(value * float64(multiplier))
}
