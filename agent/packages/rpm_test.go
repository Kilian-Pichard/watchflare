package packages

import (
	"testing"
	"time"
)

func TestParseRpmLine(t *testing.T) {
	tests := []struct {
		input       string
		wantNil     bool
		name        string
		version     string
		arch        string
		description string
		wantSize    int64
		wantDate    bool
	}{
		{
			// Full line with all fields
			input:       "bash|5.1.8-6.el9|x86_64|7372338|1697123456|The GNU Bourne Again shell",
			name:        "bash",
			version:     "5.1.8-6.el9",
			arch:        "x86_64",
			wantSize:    7372338,
			description: "The GNU Bourne Again shell",
			wantDate:    true,
		},
		{
			// Zero install time
			input:    "gpg-pubkey|a15703c1-62c807c9|(none)|0|0|gpg(Red Hat)",
			name:     "gpg-pubkey",
			version:  "a15703c1-62c807c9",
			arch:     "(none)",
			wantSize: 0,
			wantDate: false,
		},
		{
			// Empty line
			input:   "",
			wantNil: true,
		},
		{
			// Too few fields
			input:   "onlyname|1.0.0",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseRpmLine(tt.input)
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
			if pkg.Architecture != tt.arch {
				t.Errorf("architecture: got %q, want %q", pkg.Architecture, tt.arch)
			}
			if pkg.PackageManager != "rpm" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "rpm")
			}
			if pkg.PackageSize != tt.wantSize {
				t.Errorf("size: got %d, want %d", pkg.PackageSize, tt.wantSize)
			}
			if tt.description != "" && pkg.Description != tt.description {
				t.Errorf("description: got %q, want %q", pkg.Description, tt.description)
			}
			if tt.wantDate && pkg.InstalledAt.Equal(time.Time{}) {
				t.Error("expected non-zero install date")
			}
			if !tt.wantDate && !pkg.InstalledAt.Equal(time.Time{}) {
				t.Errorf("expected zero install date, got %v", pkg.InstalledAt)
			}
		})
	}
}
