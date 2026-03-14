package packages

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MavenCollector collects packages from Maven local repository
type MavenCollector struct{}

// Name returns the collector name
func (m *MavenCollector) Name() string {
	return "maven"
}

// IsAvailable checks if Maven local repository exists
func (m *MavenCollector) IsAvailable() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	m2Dir := filepath.Join(homeDir, ".m2", "repository")
	_, err = os.Stat(m2Dir)
	return err == nil
}

// Collect gathers all packages from Maven local repository
func (m *MavenCollector) Collect() ([]*Package, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	m2Repo := filepath.Join(homeDir, ".m2", "repository")

	var packages []*Package

	// Walk through the repository directory
	// Maven structure: groupId/artifactId/version/
	err = filepath.Walk(m2Repo, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for .pom files which indicate a Maven artifact
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pom") {
			pkg := m.parsePomFile(path, m2Repo)
			if pkg != nil {
				packages = append(packages, pkg)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking maven repository: %w", err)
	}

	return packages, nil
}

// MavenPom represents a Maven POM file structure
type MavenPom struct {
	XMLName    xml.Name `xml:"project"`
	GroupId    string   `xml:"groupId"`
	ArtifactId string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Name       string   `xml:"name"`
	Parent     struct {
		GroupId    string `xml:"groupId"`
		ArtifactId string `xml:"artifactId"`
		Version    string `xml:"version"`
	} `xml:"parent"`
}

// parsePomFile parses a POM file and creates a Package
func (m *MavenCollector) parsePomFile(pomPath, repoRoot string) *Package {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil
	}

	var pom MavenPom
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil
	}

	// Use parent values if direct values are empty
	groupId := pom.GroupId
	if groupId == "" && pom.Parent.GroupId != "" {
		groupId = pom.Parent.GroupId
	}

	version := pom.Version
	if version == "" && pom.Parent.Version != "" {
		version = pom.Parent.Version
	}

	if pom.ArtifactId == "" || groupId == "" || version == "" {
		return nil
	}

	// Construct full name: groupId:artifactId
	fullName := fmt.Sprintf("%s:%s", groupId, pom.ArtifactId)

	// Get file info for timestamp
	info, err := os.Stat(pomPath)
	var modTime time.Time
	if err == nil {
		modTime = info.ModTime()
	}

	// Calculate directory size (all files for this artifact version)
	artifactDir := filepath.Dir(pomPath)
	dirSize := m.calculateDirSize(artifactDir)

	description := pom.Name
	if description == "" {
		description = pom.ArtifactId
	}

	return &Package{
		Name:           fullName,
		Version:        version,
		Architecture:   "",
		PackageManager: "maven",
		Source:         groupId, // Use groupId as source
		InstalledAt:    modTime,
		PackageSize:    dirSize,
		Description:    description,
	}
}

// calculateDirSize calculates the total size of all files in a directory
func (m *MavenCollector) calculateDirSize(dir string) int64 {
	var size int64

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		size += info.Size()
	}

	return size
}
