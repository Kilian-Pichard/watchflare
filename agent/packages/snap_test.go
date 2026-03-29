package packages

import (
	"testing"
)

func TestParseSnapLine(t *testing.T) {
	tests := []struct {
		input       string
		wantNil     bool
		name        string
		version     string
		source      string
		description string
	}{
		{
			// Full line with all fields
			input:       "core20 20231123 2114 latest/stable canonical* base",
			name:        "core20",
			version:     "20231123",
			source:      "latest/stable",
			description: "Snap package from canonical* (rev: 2114)",
		},
		{
			// Minimal: name + version only — description should be empty
			input:       "mypkg 1.0.0",
			name:        "mypkg",
			version:     "1.0.0",
			source:      "",
			description: "",
		},
		{
			// Revision without publisher
			input:       "core 16-2.61.3 16798 latest/stable",
			name:        "core",
			version:     "16-2.61.3",
			source:      "latest/stable",
			description: "rev: 16798",
		},
		{
			// Only one field
			input:   "onlyname",
			wantNil: true,
		},
		{
			// Empty line
			input:   "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseSnapLine(tt.input)
			if tt.wantNil {
				if pkg != nil {
					t.Errorf("expected nil, got %+v", pkg)
				}
				return
			}
			if pkg == nil {
				t.Fatalf("expected package, got nil")
			}
			if pkg.Name != tt.name {
				t.Errorf("name: got %q, want %q", pkg.Name, tt.name)
			}
			if pkg.Version != tt.version {
				t.Errorf("version: got %q, want %q", pkg.Version, tt.version)
			}
			if pkg.PackageManager != "snap" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "snap")
			}
			if pkg.Source != tt.source {
				t.Errorf("source: got %q, want %q", pkg.Source, tt.source)
			}
			if pkg.Description != tt.description {
				t.Errorf("description: got %q, want %q", pkg.Description, tt.description)
			}
		})
	}
}
