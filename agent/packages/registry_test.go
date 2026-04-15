package packages

import (
	"runtime"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if len(registry.collectors) == 0 {
		t.Error("Registry has no collectors registered")
	}

	t.Logf("Total collectors registered: %d", len(registry.collectors))
}

func TestGetAvailableCollectors(t *testing.T) {
	registry := NewRegistry()
	available := registry.GetAvailableCollectors()

	t.Logf("Available collectors on this system (%s): %d", runtime.GOOS, len(available))

	for _, c := range available {
		t.Logf("  - %s", c.Name())
	}

	// We should have at least the CLI tools collector and some language collectors
	if len(available) == 0 {
		t.Error("No collectors are available")
	}
}

func TestLinuxCollectorsRegistered(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	registry := NewRegistry()

	expectedCollectors := []string{
		"dpkg",
		"rpm",
		"pacman",
		"apk",
		"zypper",
		"snap",
		"flatpak",
		"appimage",
	}

	// Check that all Linux collectors are registered
	for _, expected := range expectedCollectors {
		found := false
		for _, c := range registry.collectors {
			if c.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected collector '%s' not registered", expected)
		}
	}
}

func TestListCollectorNames(t *testing.T) {
	registry := NewRegistry()
	names := registry.ListCollectorNames()

	if len(names) == 0 {
		t.Error("ListCollectorNames() returned empty list")
	}

	t.Logf("Available collector names: %v", names)
}

func TestGetCollectorByName(t *testing.T) {
	registry := NewRegistry()

	// Test with CLI tools (should be available on all platforms)
	cliTools := registry.GetCollectorByName("cli-tools")
	if cliTools == nil {
		t.Error("Could not get cli-tools collector by name")
	} else if cliTools.Name() != "cli-tools" {
		t.Errorf("Expected 'cli-tools', got '%s'", cliTools.Name())
	}

	// Test with non-existent collector
	fake := registry.GetCollectorByName("nonexistent")
	if fake != nil {
		t.Error("GetCollectorByName should return nil for non-existent collector")
	}
}

func TestUpdateCheckersRegistered(t *testing.T) {
	registry := NewRegistry()

	if len(registry.updateCheckers) == 0 {
		t.Errorf("no update checkers registered on %s", runtime.GOOS)
	}

	t.Logf("Update checkers registered on %s: %d", runtime.GOOS, len(registry.updateCheckers))
	for _, c := range registry.updateCheckers {
		t.Logf("  - %s", c.Name())
	}
}

func TestGetAvailableUpdateCheckers(t *testing.T) {
	registry := NewRegistry()
	available := registry.GetAvailableUpdateCheckers()

	t.Logf("Available update checkers on %s: %d", runtime.GOOS, len(available))
	for _, c := range available {
		t.Logf("  - %s (covers: %v)", c.Name(), c.PackageManagers())
	}

	// Verify each returned checker reports IsAvailable()=true
	for _, c := range available {
		if !c.IsAvailable() {
			t.Errorf("checker %q returned by GetAvailableUpdateCheckers but IsAvailable() is false", c.Name())
		}
	}
}
