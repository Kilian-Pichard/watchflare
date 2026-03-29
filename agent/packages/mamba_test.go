package packages

import (
	"testing"
)

func TestParseMambaLine(t *testing.T) {
	tests := []struct {
		input       string
		wantNil     bool
		name        string
		version     string
		source      string
		description string
	}{
		{
			input:       "numpy                     1.24.3           py311h08b1b3b_0    conda-forge",
			name:        "numpy",
			version:     "1.24.3",
			source:      "conda-forge",
			description: "Channel: conda-forge, Build: py311h08b1b3b_0",
		},
		{
			// No channel
			input:       "pip                       23.0.1           py311_0",
			name:        "pip",
			version:     "23.0.1",
			source:      "",
			description: "Build: py311_0",
		},
		{
			// Only name and version
			input:       "setuptools 68.0.0",
			name:        "setuptools",
			version:     "68.0.0",
			source:      "",
			description: "Build: ",
		},
		{
			// Only one field
			input:   "onlyname",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseMambaLine(tt.input)
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
			if pkg.PackageManager != "mamba" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "mamba")
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
