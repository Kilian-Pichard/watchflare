package packages

import (
	"testing"
)

func TestParsePortLine(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		name    string
		version string
	}{
		{
			// Active package with variants
			input:   "  git @2.43.0_0+credential_osxkeychain+diff_highlight (active)",
			name:    "git",
			version: "2.43.0_0",
		},
		{
			// Active package without variants
			input:   "  openssl @3.2.0_0 (active)",
			name:    "openssl",
			version: "3.2.0_0",
		},
		{
			// Inactive version (no active marker)
			input:   "  openssl @1.1.1w_0",
			name:    "openssl",
			version: "1.1.1w_0",
		},
		{
			// Header line — not indented
			input:   "The following ports are currently installed:",
			wantNil: true,
		},
		{
			// Empty line
			input:   "",
			wantNil: true,
		},
		{
			// Blank/whitespace line
			input:   "   ",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parsePortLine(tt.input)
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
			if pkg.PackageManager != "macports" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "macports")
			}
			if pkg.Source != "macports.org" {
				t.Errorf("source: got %q, want %q", pkg.Source, "macports.org")
			}
		})
	}
}
