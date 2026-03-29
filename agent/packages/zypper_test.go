package packages

import (
	"testing"
)

func TestParseZypperLine(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		name    string
		version string
		arch    string
		source  string
	}{
		{
			// Installed package
			input:   "i | bash | package | 5.2.15-1.1 | x86_64 | openSUSE-Tumbleweed-Oss",
			name:    "bash",
			version: "5.2.15-1.1",
			arch:    "x86_64",
			source:  "openSUSE-Tumbleweed-Oss",
		},
		{
			// Not installed (status != "i") — skip
			input:   "v | bash | package | 5.2.14-1.1 | x86_64 | openSUSE-Tumbleweed-Oss",
			wantNil: true,
		},
		{
			// Non-package type (patch) — skip
			input:   "i | openssl | patch | 3.1.4 | noarch | openSUSE-Tumbleweed-Updates",
			wantNil: true,
		},
		{
			// Too few fields
			input:   "i | bash | package",
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
			pkg := parseZypperLine(tt.input)
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
			if pkg.PackageManager != "zypper" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "zypper")
			}
			if pkg.Source != tt.source {
				t.Errorf("source: got %q, want %q", pkg.Source, tt.source)
			}
		})
	}
}
