package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ApkCollector collects packages from apk (Alpine Linux)
type ApkCollector struct{}

// Name returns the collector name
func (a *ApkCollector) Name() string {
	return "apk"
}

// IsAvailable checks if apk is available
func (a *ApkCollector) IsAvailable() bool {
	_, err := exec.LookPath("apk")
	return err == nil
}

// Collect gathers all installed packages from apk
func (a *ApkCollector) Collect() ([]*Package, error) {
	// Run apk info to get installed packages
	// -v for version, -d for description, -s for size
	cmd := exec.Command("apk", "info", "-vv")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("apk info failed: %w", err)
	}

	var packages []*Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse package info
		// Format: "package-name-version - description"
		pkg := a.parsePackageLine(line)
		if pkg != nil {
			packages = append(packages, pkg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading apk output: %w", err)
	}

	return packages, nil
}

// parsePackageLine parses a single line of apk info output
func (a *ApkCollector) parsePackageLine(line string) *Package {
	// Try to split by " - " to separate package info from description
	parts := strings.SplitN(line, " - ", 2)
	packagePart := parts[0]
	description := ""
	if len(parts) > 1 {
		description = parts[1]
	}

	// Extract package name and version
	// Format: "package-name-version-release"
	// We need to find where the version starts (typically after the last alphabetic part of the name)
	fields := strings.Fields(packagePart)
	if len(fields) == 0 {
		return nil
	}

	fullName := fields[0]

	// Parse name-version from apk package format
	// Alpine packages are like: package-1.2.3-r0
	name, version := a.splitNameVersion(fullName)
	if name == "" {
		return nil
	}

	// Get detailed info if needed
	info := a.getPackageDetails(name)

	installDate := time.Time{}
	if info.InstallDate != nil {
		installDate = *info.InstallDate
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   info.Architecture,
		PackageManager: "apk",
		Source:         info.Repository,
		InstalledAt:    installDate,
		PackageSize:    info.Size,
		Description:    description,
	}
}

// splitNameVersion splits an apk package string into name and version
func (a *ApkCollector) splitNameVersion(fullName string) (string, string) {
	// Alpine packages follow pattern: name-version-release
	// Version typically starts with a digit
	// Example: "alpine-base-3.18.4-r0" -> name="alpine-base", version="3.18.4-r0"

	parts := strings.Split(fullName, "-")
	if len(parts) < 2 {
		return fullName, ""
	}

	// Find where version starts (first part that starts with a digit)
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) > 0 && parts[i][0] >= '0' && parts[i][0] <= '9' {
			// Found version start
			name := strings.Join(parts[:i], "-")
			version := strings.Join(parts[i:], "-")
			return name, version
		}
	}

	// Couldn't determine version
	return fullName, ""
}

type apkPackageDetails struct {
	Architecture string
	Repository   string
	InstallDate  *time.Time
	Size         int64
}

// getPackageDetails retrieves detailed information for a package
func (a *ApkCollector) getPackageDetails(name string) apkPackageDetails {
	details := apkPackageDetails{}

	// Run apk info with specific package name
	cmd := exec.Command("apk", "info", "-a", name)
	output, err := cmd.Output()
	if err != nil {
		return details
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "architecture:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				details.Architecture = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "installed size:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				sizeStr := strings.TrimSpace(parts[1])
				details.Size = parseApkSize(sizeStr)
			}
		}
	}

	return details
}

// parseApkSize converts apk size strings to bytes
func parseApkSize(sizeStr string) int64 {
	// Remove any trailing whitespace and units
	sizeStr = strings.TrimSpace(sizeStr)

	// APK sizes are typically in bytes as plain numbers
	// But may have units like "1.2 MiB"
	parts := strings.Fields(sizeStr)
	if len(parts) == 0 {
		return 0
	}

	value, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		// Try parsing as float with unit
		if len(parts) >= 2 {
			floatVal, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return 0
			}

			unit := strings.ToLower(parts[1])
			multiplier := int64(1)

			switch unit {
			case "kib", "k":
				multiplier = 1024
			case "mib", "m":
				multiplier = 1024 * 1024
			case "gib", "g":
				multiplier = 1024 * 1024 * 1024
			}

			return int64(floatVal * float64(multiplier))
		}
		return 0
	}

	return value
}
