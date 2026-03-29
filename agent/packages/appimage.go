package packages

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// AppImageCollector collects AppImage applications (Linux portable apps)
type AppImageCollector struct{}

// Name returns the collector name
func (a *AppImageCollector) Name() string {
	return "appimage"
}

// IsAvailable checks if we're on Linux (AppImages are Linux-only)
func (a *AppImageCollector) IsAvailable() bool {
	return runtime.GOOS == "linux"
}

// Collect gathers all AppImage files from common locations
func (a *AppImageCollector) Collect() ([]*Package, error) {
	var packages []*Package

	// Common AppImage locations
	searchPaths := []string{
		"~/Applications",
		"~/.local/bin",
		"~/bin",
		"/opt",
		"/usr/local/bin",
	}

	for _, searchPath := range searchPaths {
		// Expand home directory
		if strings.HasPrefix(searchPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			searchPath = filepath.Join(homeDir, searchPath[2:])
		}

		// Check if directory exists
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		// Find AppImage files in this directory
		appImages, err := a.findAppImages(searchPath)
		if err != nil {
			continue
		}

		packages = append(packages, appImages...)
	}

	return packages, nil
}

// findAppImages searches for AppImage files in a directory
func (a *AppImageCollector) findAppImages(dir string) ([]*Package, error) {
	var packages []*Package

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories
		if info.IsDir() {
			// Don't recurse too deep (max 2 levels)
			relPath, _ := filepath.Rel(dir, path)
			depth := strings.Count(relPath, string(os.PathSeparator))
			if depth > 1 {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is an AppImage
		if a.isAppImage(path) {
			pkg := a.createPackageFromAppImage(path, info)
			if pkg != nil {
				packages = append(packages, pkg)
			}
		}

		return nil
	})

	return packages, err
}

// isAppImage checks if a file is an AppImage based on its extension.
func (a *AppImageCollector) isAppImage(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".appimage")
}

// createPackageFromAppImage creates a Package from an AppImage file
func (a *AppImageCollector) createPackageFromAppImage(path string, info os.FileInfo) *Package {
	filename := filepath.Base(path)

	// Try to extract name and version from filename
	// Common patterns:
	// - AppName-1.2.3-x86_64.AppImage
	// - AppName-version.AppImage
	// - AppName.AppImage
	name, version := a.parseAppImageName(filename)

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   a.detectArch(filename),
		PackageManager: "appimage",
		Source:         path,           // full path to the AppImage file
		InstalledAt:    info.ModTime(), // file modification time as install proxy
		PackageSize:    info.Size(),
	}
}

// knownArchSuffixes lists all arch suffixes (with separator) to strip before parsing.
// Each entry is in lowercase; checked case-insensitively against the filename.
var knownArchSuffixes = []string{
	"-x86_64", "_x86_64",
	"-amd64", "_amd64",
	"-aarch64", "_aarch64",
	"-arm64", "_arm64",
	"-i686", "_i686",
	"-i386", "_i386",
	"-armhf", "_armhf",
	"-armv7l", "_armv7l",
	"-armv7", "_armv7",
}

// parseAppImageName extracts name and version from AppImage filename.
// It strips the .AppImage extension and any arch suffix before splitting.
func (a *AppImageCollector) parseAppImageName(filename string) (string, string) {
	// Remove .AppImage extension (case-insensitive)
	lower := strings.ToLower(filename)
	name := filename
	if strings.HasSuffix(lower, ".appimage") {
		name = filename[:len(filename)-len(".appimage")]
	}

	// Strip known arch suffixes before splitting, to avoid splitting "x86_64" into ["x86","64"].
	nameLower := strings.ToLower(name)
	for _, suffix := range knownArchSuffixes {
		if strings.HasSuffix(nameLower, suffix) {
			name = name[:len(name)-len(suffix)]
			break
		}
	}

	if name == "" {
		return filename, ""
	}

	// Split on separators (-/_) and locate the version component (first digit-leading part from end).
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	})

	if len(parts) == 0 {
		return name, ""
	}

	// Scan forward from index 1 to find where the version starts
	// (the first part that begins with a digit).
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 && parts[i][0] >= '0' && parts[i][0] <= '9' {
			return strings.Join(parts[:i], "-"), strings.Join(parts[i:], "-")
		}
	}

	// No version found
	return name, ""
}

// detectArch tries to detect architecture from filename
func (a *AppImageCollector) detectArch(filename string) string {
	lower := strings.ToLower(filename)

	if strings.Contains(lower, "x86_64") || strings.Contains(lower, "amd64") {
		return "x86_64"
	}
	if strings.Contains(lower, "i686") || strings.Contains(lower, "i386") {
		return "i686"
	}
	if strings.Contains(lower, "arm64") || strings.Contains(lower, "aarch64") {
		return "arm64"
	}
	if strings.Contains(lower, "armhf") || strings.Contains(lower, "armv7") {
		return "armhf"
	}

	return ""
}
