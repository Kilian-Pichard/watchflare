package packages

import (
	"log/slog"
	"time"
)

// Package represents an installed package on the system
type Package struct {
	Name              string
	Version           string
	Architecture      string
	PackageManager    string
	Source            string
	InstalledAt       time.Time
	PackageSize       int64
	Description       string
	AvailableVersion  string // Newer version available (empty if up to date)
	HasSecurityUpdate bool   // True if AvailableVersion is a security update
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

// CollectAll discovers and runs all available collectors using the registry,
// then enriches the results with available update information.
// Individual collector/checker failures are logged and skipped; the error return is always nil.
func CollectAll() ([]*Package, error) {
	registry := NewRegistry()

	var allPackages []*Package

	for _, collector := range registry.GetAvailableCollectors() {
		slog.Debug("collecting packages", "manager", collector.Name())
		pkgs, err := collector.Collect()
		if err != nil {
			slog.Warn("package collector failed", "manager", collector.Name(), "error", err)
			continue
		}
		slog.Debug("packages collected", "manager", collector.Name(), "count", len(pkgs))
		allPackages = append(allPackages, pkgs...)
	}

	// Enrich packages with update availability.
	availableCheckers := registry.GetAvailableUpdateCheckers()
	for _, checker := range availableCheckers {
		slog.Debug("checking updates", "checker", checker.Name())
		updates, err := checker.CheckUpdates()
		if err != nil {
			slog.Warn("update checker failed", "checker", checker.Name(), "error", err)
			continue
		}
		slog.Debug("update checker results", "checker", checker.Name(), "updates_found", len(updates))
		if len(updates) == 0 {
			continue
		}

		// Build a set of package managers this checker covers for fast lookup
		pmSet := make(map[string]bool, len(checker.PackageManagers()))
		for _, pm := range checker.PackageManagers() {
			pmSet[pm] = true
		}

		outdated := 0
		for _, pkg := range allPackages {
			if !pmSet[pkg.PackageManager] {
				continue
			}
			if status, ok := updates[pkg.Name]; ok {
				pkg.AvailableVersion = status.AvailableVersion
				pkg.HasSecurityUpdate = status.HasSecurityUpdate
				outdated++
			}
		}
		slog.Debug("update check complete", "checker", checker.Name(), "outdated", outdated)
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
