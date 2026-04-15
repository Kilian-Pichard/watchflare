package packages

import (
	"testing"
)

func TestParseGemLine(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		name    string
		version string
	}{
		{
			input:   "bundler (2.4.10)",
			name:    "bundler",
			version: "2.4.10",
		},
		{
			// Multiple versions — only first reported
			input:   "rake (13.0.6, 12.3.3)",
			name:    "rake",
			version: "13.0.6",
		},
		{
			// Version with platform suffix
			input:   "nokogiri (1.15.4 x86_64-linux)",
			name:    "nokogiri",
			version: "1.15.4 x86_64-linux",
		},
		{
			// Default gem bundled with Ruby
			input:   "io-console (default: 0.6.0)",
			name:    "io-console",
			version: "0.6.0",
		},
		{
			// Default gem with multiple versions
			input:   "json (default: 2.6.3, 2.5.1)",
			name:    "json",
			version: "2.6.3",
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
		{
			// No version parentheses
			input:   "somegem",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseGemLine(tt.input)
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
			if pkg.PackageManager != "gem" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "gem")
			}
			if pkg.Source != "rubygems.org" {
				t.Errorf("source: got %q, want %q", pkg.Source, "rubygems.org")
			}
		})
	}
}
