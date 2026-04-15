package packages

// UpdateStatus represents available update information for a single package
type UpdateStatus struct {
	AvailableVersion  string
	HasSecurityUpdate bool
}

// UpdateChecker checks which installed packages have available updates
type UpdateChecker interface {
	// Name returns the checker identifier (e.g., "apt", "dnf", "checkupdates")
	Name() string

	// IsAvailable checks if the underlying tool is available on the system
	IsAvailable() bool

	// PackageManagers returns the package managers this checker covers (e.g., ["dpkg"])
	PackageManagers() []string

	// CheckUpdates returns update status keyed by package name, for packages that have
	// an available update. Packages without updates are not included in the map.
	CheckUpdates() (map[string]UpdateStatus, error)
}
