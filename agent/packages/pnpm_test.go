package packages

import (
	"testing"
)

func TestParsePnpmLine(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		name    string
		version string
	}{
		{
			// Standard "name version" format
			input:   "typescript 5.3.3",
			name:    "typescript",
			version: "5.3.3",
		},
		{
			// Scoped package "@scope/name version"
			input:   "@angular/cli 17.1.0",
			name:    "@angular/cli",
			version: "17.1.0",
		},
		{
			// Tree-decorated line "├── name version"
			input:   "├── typescript 5.3.3",
			name:    "typescript",
			version: "5.3.3",
		},
		{
			// "name@version" format
			input:   "pnpm@8.15.1",
			name:    "pnpm",
			version: "8.15.1",
		},
		{
			// Header line — skip
			input:   "dependencies:",
			wantNil: true,
		},
		{
			// Path line — skip
			input:   "/usr/local/lib",
			wantNil: true,
		},
		{
			// Legend line — skip
			input:   "Legend: production dependency",
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
		{
			// Name only, no version
			input:   "mypackage",
			name:    "mypackage",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parsePnpmLine(tt.input)
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
			if pkg.PackageManager != "pnpm-global" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "pnpm-global")
			}
			if pkg.Source != "npmjs.com" {
				t.Errorf("source: got %q, want %q", pkg.Source, "npmjs.com")
			}
		})
	}
}
