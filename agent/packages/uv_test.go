package packages

import (
	"testing"
)

func TestParseUvLine(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		name    string
		version string
	}{
		{
			// Standard "name vVersion" format
			input:   "ruff v0.2.0",
			name:    "ruff",
			version: "0.2.0",
		},
		{
			// Version without "v" prefix
			input:   "black 23.12.1",
			name:    "black",
			version: "23.12.1",
		},
		{
			// "name@version" format
			input:   "mypy@1.8.0",
			name:    "mypy",
			version: "1.8.0",
		},
		{
			// Name only, no version
			input:   "pytest",
			name:    "pytest",
			version: "",
		},
		{
			// Sub-command line (indented "- cmd") — skip
			input:   "    - ruff",
			wantNil: true,
		},
		{
			// After trim, starts with "-" — skip
			input:   "- black",
			wantNil: true,
		},
		{
			// "No tools installed" message — skip
			input:   "No tools installed",
			wantNil: true,
		},
		{
			// Empty line
			input:   "",
			wantNil: true,
		},
		{
			// Whitespace only
			input:   "   ",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseUvLine(tt.input)
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
			if pkg.PackageManager != "uv" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "uv")
			}
			if pkg.Source != "pypi.org" {
				t.Errorf("source: got %q, want %q", pkg.Source, "pypi.org")
			}
		})
	}
}
