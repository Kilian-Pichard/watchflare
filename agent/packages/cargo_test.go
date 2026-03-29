package packages

import (
	"testing"
)

func TestCargoCollect_ParsesOutput(t *testing.T) {
	output := []byte(`ripgrep v13.0.0:
    ripgrep
fd-find v9.0.0:
    fd
cargo-edit v0.12.2:
    cargo-add
    cargo-rm
    cargo-upgrade
`)
	pkgs := parseCargoOutput(output)
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}

	tests := []struct{ name, version string }{
		{"ripgrep", "13.0.0"},
		{"fd-find", "9.0.0"},
		{"cargo-edit", "0.12.2"},
	}

	for i, tt := range tests {
		if pkgs[i].Name != tt.name {
			t.Errorf("[%d] name: got %q, want %q", i, pkgs[i].Name, tt.name)
		}
		if pkgs[i].Version != tt.version {
			t.Errorf("[%d] version: got %q, want %q", i, pkgs[i].Version, tt.version)
		}
		if pkgs[i].PackageManager != "cargo" {
			t.Errorf("[%d] package manager: got %q, want %q", i, pkgs[i].PackageManager, "cargo")
		}
	}
}

func TestCargoCollect_EmptyOutput(t *testing.T) {
	pkgs := parseCargoOutput([]byte(""))
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestCargoCollect_SkipsBinaryLines(t *testing.T) {
	output := []byte(`ripgrep v13.0.0:
    ripgrep
    rg
`)
	pkgs := parseCargoOutput(output)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "ripgrep" {
		t.Errorf("expected %q, got %q", "ripgrep", pkgs[0].Name)
	}
}
