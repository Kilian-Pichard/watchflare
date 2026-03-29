package packages

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const mavenRepoPath = ".m2/repository"

// MavenCollector collects packages from Maven local repository
type MavenCollector struct {
	repoPath string
}

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
	repoPath := filepath.Join(homeDir, mavenRepoPath)
	if _, err := os.Stat(repoPath); err != nil {
		return false
	}
	m.repoPath = repoPath
	return true
}

// Collect gathers all packages from Maven local repository.
// Walks ~/.m2/repository and parses each .pom file.
func (m *MavenCollector) Collect() ([]*Package, error) {
	var packages []*Package

	err := filepath.Walk(m.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pom") {
			if pkg := parsePomFile(path); pkg != nil {
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

// mavenPom represents a Maven POM file structure
type mavenPom struct {
	XMLName    xml.Name `xml:"project"`
	GroupId    string   `xml:"groupId"`
	ArtifactId string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Name       string   `xml:"name"`
	Parent     struct {
		GroupId string `xml:"groupId"`
		Version string `xml:"version"`
	} `xml:"parent"`
}

// parsePomFile reads a .pom file from disk and returns a Package.
func parsePomFile(pomPath string) *Package {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil
	}

	groupID, artifactID, version, description, ok := parseMavenPomData(data)
	if !ok {
		return nil
	}

	var installedAt time.Time
	if info, err := os.Stat(pomPath); err == nil {
		installedAt = info.ModTime()
	}

	return &Package{
		Name:           fmt.Sprintf("%s:%s", groupID, artifactID),
		Version:        version,
		PackageManager: "maven",
		Source:         groupID,
		InstalledAt:    installedAt,
		PackageSize:    calculateDirSize(filepath.Dir(pomPath)),
		Description:    description,
	}
}

// parseMavenPomData parses POM XML bytes and resolves groupId/version from parent
// when not set directly. Returns ok=false if required fields are missing.
func parseMavenPomData(data []byte) (groupID, artifactID, version, description string, ok bool) {
	var pom mavenPom
	if err := xml.Unmarshal(data, &pom); err != nil {
		return
	}

	groupID = pom.GroupId
	if groupID == "" {
		groupID = pom.Parent.GroupId
	}
	version = pom.Version
	if version == "" {
		version = pom.Parent.Version
	}
	artifactID = pom.ArtifactId

	if groupID == "" || artifactID == "" || version == "" {
		return
	}

	description = pom.Name
	if description == "" {
		description = artifactID
	}

	ok = true
	return
}

// calculateDirSize returns the total size of all files (non-recursive) in dir.
func calculateDirSize(dir string) int64 {
	var size int64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if info, err := entry.Info(); err == nil {
			size += info.Size()
		}
	}
	return size
}
