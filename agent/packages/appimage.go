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
		"~/Downloads", // Many users keep AppImages in Downloads
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
		if a.isAppImage(path, info) {
			pkg := a.createPackageFromAppImage(path, info)
			if pkg != nil {
				packages = append(packages, pkg)
			}
		}

		return nil
	})

	return packages, err
}

// isAppImage checks if a file is an AppImage
func (a *AppImageCollector) isAppImage(path string, info os.FileInfo) bool {
	// Check file extension
	if strings.HasSuffix(strings.ToLower(path), ".appimage") {
		return true
	}

	// Check if executable and starts with AppImage magic bytes
	// AppImages are ELF executables with specific magic
	if info.Mode()&0111 != 0 { // Is executable
		// Could read first bytes to check for ELF + AppImage magic
		// For now, just check extension
		return false
	}

	return false
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
		Source:         filepath.Dir(path), // Store the directory
		InstalledAt:    info.ModTime(),     // Use file modification time
		PackageSize:    info.Size(),
		Description:    path, // Store full path
	}
}

// parseAppImageName extracts name and version from AppImage filename
func (a *AppImageCollector) parseAppImageName(filename string) (string, string) {
	// Remove .AppImage extension
	name := strings.TrimSuffix(filename, ".AppImage")
	name = strings.TrimSuffix(name, ".appimage")

	// Common patterns:
	// - Name-version-arch
	// - Name-version
	// - Name_version

	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	})

	if len(parts) == 0 {
		return name, ""
	}

	// Last part might be architecture
	lastPart := parts[len(parts)-1]
	if a.isArchString(lastPart) {
		parts = parts[:len(parts)-1]
	}

	if len(parts) == 0 {
		return name, ""
	}

	// Second-to-last part might be version
	for i := len(parts) - 1; i > 0; i-- {
		part := parts[i]
		// Check if this looks like a version (starts with digit)
		if len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
			appName := strings.Join(parts[:i], "-")
			version := strings.Join(parts[i:], "-")
			return appName, version
		}
	}

	// No version found
	return name, ""
}

// isArchString checks if a string looks like an architecture
func (a *AppImageCollector) isArchString(s string) bool {
	archStrings := []string{"x86_64", "i686", "amd64", "arm64", "aarch64", "armhf"}
	s = strings.ToLower(s)
	for _, arch := range archStrings {
		if s == arch {
			return true
		}
	}
	return false
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
