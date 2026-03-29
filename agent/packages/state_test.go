package packages

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- LoadState ---

func TestLoadState_FileNotExist(t *testing.T) {
	state, err := LoadState(filepath.Join(t.TempDir(), "notexist.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if len(state.Packages) != 0 {
		t.Errorf("expected empty packages, got %d", len(state.Packages))
	}
}

func TestLoadState_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	state := &PackageState{
		LastScan:     time.Now().Truncate(time.Second),
		PackageCount: 1,
		Packages: []*Package{
			{Name: "curl", Version: "8.5.0", PackageManager: "dpkg"},
		},
	}
	if err := state.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(loaded.Packages))
	}
	if loaded.Packages[0].Name != "curl" {
		t.Errorf("name: got %q, want %q", loaded.Packages[0].Name, "curl")
	}
	if loaded.Packages[0].Version != "8.5.0" {
		t.Errorf("version: got %q, want %q", loaded.Packages[0].Version, "8.5.0")
	}
}

func TestLoadState_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("not json"), 0640); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Corrupted file → empty state, no error
	state, err := LoadState(path)
	if err != nil {
		t.Fatalf("unexpected error for corrupted file: %v", err)
	}
	if len(state.Packages) != 0 {
		t.Errorf("expected empty packages after corruption, got %d", len(state.Packages))
	}
}

// --- Save ---

func TestSave_CreatesDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subdir", "state.json")
	state := &PackageState{Packages: []*Package{}}
	if err := state.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestSave_SyncsPackageCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state := &PackageState{
		PackageCount: 99, // intentionally wrong
		Packages: []*Package{
			{Name: "curl", Version: "8.5.0", PackageManager: "dpkg"},
			{Name: "git", Version: "2.43.0", PackageManager: "dpkg"},
		},
	}
	if err := state.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.PackageCount != 2 {
		t.Errorf("PackageCount: got %d, want 2 (auto-synced from len(Packages))", loaded.PackageCount)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state := &PackageState{Packages: []*Package{}}
	if err := state.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("permissions: got %o, want %o", info.Mode().Perm(), 0640)
	}
}

// --- ComputeDelta ---

func TestComputeDelta_Added(t *testing.T) {
	state := &PackageState{Packages: []*Package{
		{Name: "curl", Version: "8.0.0", PackageManager: "dpkg"},
	}}
	newPkgs := []*Package{
		{Name: "curl", Version: "8.0.0", PackageManager: "dpkg"},
		{Name: "git", Version: "2.43.0", PackageManager: "dpkg"},
	}

	added, removed, updated := state.ComputeDelta(newPkgs)
	if len(added) != 1 || added[0].Name != "git" {
		t.Errorf("added: expected [git], got %v", added)
	}
	if len(removed) != 0 {
		t.Errorf("removed: expected none, got %v", removed)
	}
	if len(updated) != 0 {
		t.Errorf("updated: expected none, got %v", updated)
	}
}

func TestComputeDelta_Removed(t *testing.T) {
	state := &PackageState{Packages: []*Package{
		{Name: "curl", Version: "8.0.0", PackageManager: "dpkg"},
		{Name: "git", Version: "2.43.0", PackageManager: "dpkg"},
	}}
	newPkgs := []*Package{
		{Name: "curl", Version: "8.0.0", PackageManager: "dpkg"},
	}

	added, removed, updated := state.ComputeDelta(newPkgs)
	if len(added) != 0 {
		t.Errorf("added: expected none, got %v", added)
	}
	if len(removed) != 1 || removed[0].Name != "git" {
		t.Errorf("removed: expected [git], got %v", removed)
	}
	if len(updated) != 0 {
		t.Errorf("updated: expected none, got %v", updated)
	}
}

func TestComputeDelta_Updated(t *testing.T) {
	state := &PackageState{Packages: []*Package{
		{Name: "curl", Version: "8.0.0", PackageManager: "dpkg"},
	}}
	newPkgs := []*Package{
		{Name: "curl", Version: "8.5.0", PackageManager: "dpkg"},
	}

	added, removed, updated := state.ComputeDelta(newPkgs)
	if len(added) != 0 {
		t.Errorf("added: expected none, got %v", added)
	}
	if len(removed) != 0 {
		t.Errorf("removed: expected none, got %v", removed)
	}
	if len(updated) != 1 || updated[0].Version != "8.5.0" {
		t.Errorf("updated: expected [curl 8.5.0], got %v", updated)
	}
}

func TestComputeDelta_SameManagerDifferentPackages(t *testing.T) {
	// Same name, different package managers → different keys
	state := &PackageState{Packages: []*Package{
		{Name: "git", Version: "2.43.0", PackageManager: "dpkg"},
	}}
	newPkgs := []*Package{
		{Name: "git", Version: "2.43.0", PackageManager: "brew-formula"},
	}

	added, removed, _ := state.ComputeDelta(newPkgs)
	if len(added) != 1 {
		t.Errorf("added: expected 1 (brew git), got %d", len(added))
	}
	if len(removed) != 1 {
		t.Errorf("removed: expected 1 (dpkg git), got %d", len(removed))
	}
}

func TestComputeDelta_Empty(t *testing.T) {
	state := &PackageState{Packages: []*Package{}}
	added, removed, updated := state.ComputeDelta([]*Package{})
	if len(added) != 0 || len(removed) != 0 || len(updated) != 0 {
		t.Error("expected no changes for empty→empty")
	}
}

// --- HasChanges ---

func TestHasChanges(t *testing.T) {
	pkg := &Package{Name: "test"}
	if HasChanges(nil, nil, nil) {
		t.Error("expected false for all nil")
	}
	if !HasChanges([]*Package{pkg}, nil, nil) {
		t.Error("expected true when added is non-empty")
	}
	if !HasChanges(nil, []*Package{pkg}, nil) {
		t.Error("expected true when removed is non-empty")
	}
	if !HasChanges(nil, nil, []*Package{pkg}) {
		t.Error("expected true when updated is non-empty")
	}
}
