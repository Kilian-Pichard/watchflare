package packages

import (
	"testing"
)

func TestParseNpmOutput(t *testing.T) {
	output := []byte(`{
		"dependencies": {
			"typescript": {"version": "5.3.3"},
			"yarn": {"version": "1.22.21"}
		}
	}`)

	pkgs, err := parseNpmOutput(output)
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
		{"typescript", "5.3.3"},
		{"yarn", "1.22.21"},
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
		if p.PackageManager != "npm" {
			t.Errorf("%s package manager: got %q, want %q", tt.name, p.PackageManager, "npm")
		}
		if p.Source != "npmjs.com" {
			t.Errorf("%s source: got %q, want %q", tt.name, p.Source, "npmjs.com")
		}
	}
}

func TestParseNpmOutput_Empty(t *testing.T) {
	output := []byte(`{"dependencies": {}}`)

	pkgs, err := parseNpmOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParseNpmOutput_NoDependenciesKey(t *testing.T) {
	// npm list with nothing installed returns just {}
	output := []byte(`{}`)

	pkgs, err := parseNpmOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParseNpmOutput_InvalidJSON(t *testing.T) {
	_, err := parseNpmOutput([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
