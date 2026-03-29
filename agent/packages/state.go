package packages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PackageState represents the stored state of packages
type PackageState struct {
	LastScan     time.Time  `json:"last_scan"`
	PackageCount int        `json:"package_count"`
	Packages     []*Package `json:"packages"`
}

// LoadState loads the package state from a JSON file
// Returns an empty state if the file doesn't exist
func LoadState(path string) (*PackageState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return empty state (first run)
			return &PackageState{
				Packages: make([]*Package, 0),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state PackageState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupted state file - return empty state
		return &PackageState{
			Packages: make([]*Package, 0),
		}, nil
	}

	return &state, nil
}

// Save writes the package state to a JSON file.
// PackageCount is automatically synced with len(Packages) before writing.
func (s *PackageState) Save(path string) error {
	s.PackageCount = len(s.Packages)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, 0640); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// ComputeDelta compares current packages with new packages and returns changes
// Returns: added, removed, updated packages
func (s *PackageState) ComputeDelta(newPackages []*Package) (added, removed, updated []*Package) {
	// Build maps for fast lookup
	oldMap := make(map[string]*Package)
	newMap := make(map[string]*Package)

	// Map old packages by key (name|package_manager)
	for _, pkg := range s.Packages {
		key := packageKey(pkg)
		oldMap[key] = pkg
	}

	// Map new packages by key
	for _, pkg := range newPackages {
		key := packageKey(pkg)
		newMap[key] = pkg
	}

	// Find added and updated packages
	for key, newPkg := range newMap {
		if oldPkg, exists := oldMap[key]; exists {
			// Package exists - check if version changed
			if oldPkg.Version != newPkg.Version {
				updated = append(updated, newPkg)
			}
			// If version is same, no change needed
		} else {
			// New package
			added = append(added, newPkg)
		}
	}

	// Find removed packages
	for key, oldPkg := range oldMap {
		if _, exists := newMap[key]; !exists {
			removed = append(removed, oldPkg)
		}
	}

	return added, removed, updated
}

// HasChanges returns true if there are any changes (added/removed/updated)
func HasChanges(added, removed, updated []*Package) bool {
	return len(added) > 0 || len(removed) > 0 || len(updated) > 0
}

// packageKey generates a unique key for a package
// Format: "name|package_manager"
func packageKey(pkg *Package) string {
	return pkg.Name + "|" + pkg.PackageManager
}
