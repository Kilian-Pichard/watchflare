package packages

import (
	"testing"
)

func TestParseDpkgLine(t *testing.T) {
	tests := []struct {
		input       string
		wantNil     bool
		name        string
		version     string
		arch        string
		size        int64
		description string
	}{
		{
			input:       "bash|5.1-6ubuntu1|amd64|1792|ii |The GNU Bourne Again shell",
			name:        "bash",
			version:     "5.1-6ubuntu1",
			arch:        "amd64",
			size:        1792 * 1024,
			description: "The GNU Bourne Again shell",
		},
		{
			// Size 0 when field is empty
			input:       "libc6|2.35-0ubuntu3|amd64||ii |GNU C Library",
			name:        "libc6",
			version:     "2.35-0ubuntu3",
			arch:        "amd64",
			size:        0,
			description: "GNU C Library",
		},
		{
			// Not installed (rc = removed, config files remain)
			input:    "oldpkg|1.0|amd64|512|rc |Old package",
			wantNil:  true,
		},
		{
			// Not installed (un = unknown)
			input:    "ghost|2.0|amd64|0|un |Ghost package",
			wantNil:  true,
		},
		{
			// Empty line
			input:   "",
			wantNil: true,
		},
		{
			// Too few fields
			input:   "bash|5.1",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseDpkgLine(tt.input)
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
				t.Errorf("arch: got %q, want %q", pkg.Architecture, tt.arch)
			}
			if pkg.PackageSize != tt.size {
				t.Errorf("size: got %d, want %d", pkg.PackageSize, tt.size)
			}
			if pkg.PackageManager != "dpkg" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "dpkg")
			}
			if pkg.Description != tt.description {
				t.Errorf("description: got %q, want %q", pkg.Description, tt.description)
			}
		})
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1792", 1792},
		{"0", 0},
		{"", 0},
		{"  512  ", 512},
		{"notanumber", 0},
		{"-1", -1},
	}

	for _, tt := range tests {
		result := parseInt64(tt.input)
		if result != tt.expected {
			t.Errorf("parseInt64(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
