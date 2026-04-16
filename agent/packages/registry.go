package packages

import (
	"runtime"
)

// CollectorRegistry manages all available package collectors and update checkers
type CollectorRegistry struct {
	collectors     []Collector
	updateCheckers []UpdateChecker
}

// NewRegistry creates a new collector registry with all available collectors and update checkers
func NewRegistry() *CollectorRegistry {
	registry := &CollectorRegistry{
		collectors:     []Collector{},
		updateCheckers: []UpdateChecker{},
	}

	// Register platform-specific collectors
	registry.registerPlatformCollectors()

	// Register cross-platform language collectors
	registry.registerLanguageCollectors()

	// Register update checkers for installed package managers
	registry.registerUpdateCheckers()
	registry.registerLanguageUpdateCheckers()

	return registry
}

// registerPlatformCollectors registers OS-level package managers based on the current platform
func (r *CollectorRegistry) registerPlatformCollectors() {
	switch runtime.GOOS {
	case "darwin": // macOS
		r.collectors = append(r.collectors,
			&BrewCollector{},
			&MacPortsCollector{},
			&MacOSAppsCollector{},    // All macOS applications
			&MacOSPkgutilCollector{}, // System packages (.pkg)
		)
	case "linux":
		r.collectors = append(r.collectors,
			// Distribution package managers
			&DpkgCollector{},   // Debian/Ubuntu (apt)
			&RpmCollector{},    // RedHat/CentOS/Fedora (yum/dnf)
			&PacmanCollector{}, // Arch Linux
			&ApkCollector{},    // Alpine Linux
			&ZypperCollector{}, // openSUSE
			&BrewCollector{},   // Homebrew on Linux

			// Universal package managers
			&SnapCollector{},    // Snap packages (Ubuntu and others)
			&FlatpakCollector{}, // Flatpak (cross-distribution)
			&AppImageCollector{}, // AppImage portable apps
		)
	case "windows":
		// Chocolatey, Scoop, Winget will be added later
	}
}

// registerLanguageCollectors registers language-specific package managers (cross-platform)
func (r *CollectorRegistry) registerLanguageCollectors() {
	// These collectors are cross-platform
	// They self-disable if the binary doesn't exist
	r.collectors = append(r.collectors,
		// Primary package managers
		&NpmCollector{},      // Node.js
		&PipCollector{},      // Python
		&GemCollector{},      // Ruby
		&CargoCollector{},    // Rust
		&ComposerCollector{}, // PHP

		// Alternative Node.js package managers
		&YarnCollector{}, // Yarn global packages
		&PnpmCollector{}, // pnpm global packages

		// Python environment managers
		&PoetryCollector{}, // Poetry virtualenvs
		&PipxCollector{},   // pipx isolated applications
		&UvCollector{},     // uv (modern ultra-fast Python package manager)
		&CondaCollector{},  // conda (data science packages)
		&MambaCollector{},  // mamba (fast conda replacement)

		// .NET and Java
		&NuGetCollector{}, // .NET global tools
		&MavenCollector{}, // Maven local repository

		// Universal package managers
		&NixCollector{},      // Nix package manager (multi-platform)
		&CLIToolsCollector{}, // Important CLI tools (docker, kubectl, etc.)
	)
}

// GetAvailableCollectors returns only collectors that are available on this system
func (r *CollectorRegistry) GetAvailableCollectors() []Collector {
	available := make([]Collector, 0, len(r.collectors))
	for _, c := range r.collectors {
		if c.IsAvailable() {
			available = append(available, c)
		}
	}
	return available
}

// GetCollectorByName returns a specific collector by name, or nil if not found
func (r *CollectorRegistry) GetCollectorByName(name string) Collector {
	for _, c := range r.collectors {
		if c.Name() == name && c.IsAvailable() {
			return c
		}
	}
	return nil
}

// ListCollectorNames returns names of all available collectors
func (r *CollectorRegistry) ListCollectorNames() []string {
	available := r.GetAvailableCollectors()
	names := make([]string, 0, len(available))
	for _, c := range available {
		names = append(names, c.Name())
	}
	return names
}

// registerUpdateCheckers registers update checkers for the current platform.
// Each checker self-disables via IsAvailable() if the underlying tool is not installed.
func (r *CollectorRegistry) registerUpdateCheckers() {
	switch runtime.GOOS {
	case "linux":
		r.updateCheckers = append(r.updateCheckers,
			&AptUpdateChecker{},    // Debian/Ubuntu
			&DnfUpdateChecker{},    // RHEL/CentOS/Rocky/Fedora
			&PacmanUpdateChecker{}, // Arch Linux (requires pacman-contrib)
			&ApkUpdateChecker{},    // Alpine Linux
			&BrewUpdateChecker{},   // Homebrew on Linux
		)
	case "darwin":
		r.updateCheckers = append(r.updateCheckers,
			&BrewUpdateChecker{}, // Homebrew on macOS
		)
	}
}

// registerLanguageUpdateCheckers registers update checkers for language package managers.
// These are cross-platform and self-disable via IsAvailable() if the tool is not installed.
func (r *CollectorRegistry) registerLanguageUpdateCheckers() {
	r.updateCheckers = append(r.updateCheckers,
		&NpmUpdateChecker{},
		&PipUpdateChecker{},
		&GemUpdateChecker{},
		&ComposerUpdateChecker{},
	)
}

// GetAvailableUpdateCheckers returns only update checkers whose tools are installed
func (r *CollectorRegistry) GetAvailableUpdateCheckers() []UpdateChecker {
	available := make([]UpdateChecker, 0, len(r.updateCheckers))
	for _, c := range r.updateCheckers {
		if c.IsAvailable() {
			available = append(available, c)
		}
	}
	return available
}
