package packages

import (
	"log"
	"time"
)

// Package represents an installed package on the system
type Package struct {
	Name          string
	Version       string
	Architecture  string
	PackageManager string
	Source        string
	InstalledAt   time.Time
	PackageSize   int64
	Description   string
}

// Collector interface for different package managers
type Collector interface {
	// Name returns the collector name (e.g., "dpkg", "rpm", "brew")
	Name() string

	// IsAvailable checks if this package manager is available on the system
	IsAvailable() bool

	// Collect gathers all installed packages
	// Returns the list of packages or an error
	Collect() ([]*Package, error)
}

// CollectAll discovers and runs all available collectors
// Returns combined list of packages from all package managers
func CollectAll() ([]*Package, error) {
	collectors := []Collector{
		&DpkgCollector{},
		&RpmCollector{},
		&BrewCollector{},
	}

	var allPackages []*Package

	for _, collector := range collectors {
		if !collector.IsAvailable() {
			log.Printf("Package manager %s not available, skipping", collector.Name())
			continue
		}

		log.Printf("Collecting packages from %s...", collector.Name())
		packages, err := collector.Collect()
		if err != nil {
			log.Printf("Warning: %s collector failed: %v", collector.Name(), err)
			continue
		}

		log.Printf("✓ Collected %d packages from %s", len(packages), collector.Name())
		allPackages = append(allPackages, packages...)
	}

	return allPackages, nil
}

// TruncateDescription truncates description to max 100 characters
func TruncateDescription(desc string) string {
	if len(desc) <= 100 {
		return desc
	}
	return desc[:97] + "..."
}
