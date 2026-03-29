package packages

import (
	"testing"
)

func TestParseNuGetLine(t *testing.T) {
	tests := []struct {
		input       string
		wantNil     bool
		name        string
		version     string
		description string
	}{
		{
			// Full line with commands
			input:       "dotnet-ef         7.0.0      dotnet-ef",
			name:        "dotnet-ef",
			version:     "7.0.0",
			description: "dotnet-ef",
		},
		{
			// Multiple commands
			input:       "dotnet-format     7.0.0      dotnet-format dotnet-fmt",
			name:        "dotnet-format",
			version:     "7.0.0",
			description: "dotnet-format, dotnet-fmt",
		},
		{
			// No commands column
			input:       "mytool 1.2.3",
			name:        "mytool",
			version:     "1.2.3",
			description: "",
		},
		{
			// Too few fields
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
			pkg := parseNuGetLine(tt.input)
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
			if pkg.PackageManager != "nuget-global" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "nuget-global")
			}
			if pkg.Source != "nuget.org" {
				t.Errorf("source: got %q, want %q", pkg.Source, "nuget.org")
			}
			if pkg.Description != tt.description {
				t.Errorf("description: got %q, want %q", pkg.Description, tt.description)
			}
		})
	}
}
