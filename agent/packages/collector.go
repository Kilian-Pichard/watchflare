package packages

import (
	"log/slog"
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

// CollectAll discovers and runs all available collectors using the registry.
// Returns combined list of packages from all package managers.
// Individual collector failures are logged and skipped; the error return is always nil.
func CollectAll() ([]*Package, error) {
	// Create registry and get available collectors
	registry := NewRegistry()
	collectors := registry.GetAvailableCollectors()

	var allPackages []*Package

	for _, collector := range collectors {
		slog.Debug("collecting packages", "manager", collector.Name())
		packages, err := collector.Collect()
		if err != nil {
			slog.Warn("package collector failed", "manager", collector.Name(), "error", err)
			continue
		}

		slog.Debug("packages collected", "manager", collector.Name(), "count", len(packages))
		allPackages = append(allPackages, packages...)
	}

	return allPackages, nil
}

// TruncateDescription truncates description to max 100 runes
func TruncateDescription(desc string) string {
	runes := []rune(desc)
	if len(runes) <= 100 {
		return desc
	}
	return string(runes[:97]) + "..."
}

