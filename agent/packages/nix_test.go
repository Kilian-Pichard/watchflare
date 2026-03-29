package packages

import (
	"testing"
)

func TestSplitNixNameVersion(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
	}{
		{"firefox-121.0", "firefox", "121.0"},
		{"lib32-glibc-2.38", "lib32-glibc", "2.38"},
		{"python3-3.11.5", "python3", "3.11.5"},
		{"openssl-3.2.0", "openssl", "3.2.0"},
		// Version with multiple components after split
		{"gcc-13.2.0", "gcc", "13.2.0"},
		// Package with no version
		{"bash", "bash", ""},
		// 1password — name starts with digit but version is the last component
		{"1password-8.10.18", "1password", "8.10.18"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version := splitNixNameVersion(tt.input)
			if name != tt.name {
				t.Errorf("name: got %q, want %q", name, tt.name)
			}
			if version != tt.version {
				t.Errorf("version: got %q, want %q", version, tt.version)
			}
		})
	}
}

func TestParseNixLine(t *testing.T) {
	tests := []struct {
		input       string
		wantNil     bool
		name        string
		version     string
		description string
	}{
		{
			input:       "firefox-121.0  /nix/store/abc123-firefox-121.0",
			name:        "firefox",
			version:     "121.0",
			description: "/nix/store/abc123-firefox-121.0",
		},
		{
			// No store path
			input:       "git-2.43.0",
			name:        "git",
			version:     "2.43.0",
			description: "",
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
			// No version — name returned as-is, version empty, still valid
			input:       "bash",
			name:        "bash",
			version:     "",
			description: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseNixLine(tt.input)
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
			if pkg.PackageManager != "nix" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "nix")
			}
			if pkg.Description != tt.description {
				t.Errorf("description: got %q, want %q", pkg.Description, tt.description)
			}
		})
	}
}
