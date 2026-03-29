package packages

import (
	"testing"
)

func TestParsePipOutput(t *testing.T) {
	output := []byte(`[{"name": "numpy", "version": "1.26.3"}, {"name": "requests", "version": "2.31.0"}]`)

	pkgs, err := parsePipOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	byName := make(map[string]*Package)
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	tests := []struct{ name, version string }{
		{"numpy", "1.26.3"},
		{"requests", "2.31.0"},
	}
	for _, tt := range tests {
		p, ok := byName[tt.name]
		if !ok {
			t.Errorf("package %q not found", tt.name)
			continue
		}
		if p.Version != tt.version {
			t.Errorf("%s version: got %q, want %q", tt.name, p.Version, tt.version)
		}
		if p.PackageManager != "pip" {
			t.Errorf("%s package manager: got %q, want %q", tt.name, p.PackageManager, "pip")
		}
		if p.Source != "pypi.org" {
			t.Errorf("%s source: got %q, want %q", tt.name, p.Source, "pypi.org")
		}
	}
}

func TestParsePipOutput_Empty(t *testing.T) {
	pkgs, err := parsePipOutput([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParsePipOutput_InvalidJSON(t *testing.T) {
	_, err := parsePipOutput([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
