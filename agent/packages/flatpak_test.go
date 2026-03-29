package packages

import (
	"testing"
)

func TestParseFlatpakLine(t *testing.T) {
	tests := []struct {
		input       string
		wantNil     bool
		name        string
		version     string
		arch        string
		source      string
		description string
	}{
		{
			// Full line with all columns
			input:       "Spotify\tcom.spotify.Client\t1.2.13\tstable\tflathub\tx86_64",
			name:        "Spotify",
			version:     "1.2.13",
			arch:        "x86_64",
			source:      "flathub",
			description: "com.spotify.Client",
		},
		{
			// No friendly name — falls back to app ID
			input:       "\torg.gnome.Boxes\t42.0\tstable\tflathub\tx86_64",
			name:        "org.gnome.Boxes",
			version:     "42.0",
			arch:        "x86_64",
			source:      "flathub",
			description: "org.gnome.Boxes",
		},
		{
			// No version — falls back to branch
			input:       "GIMP\torg.gimp.GIMP\t\tstable\tflathub\tx86_64",
			name:        "GIMP",
			version:     "stable",
			arch:        "x86_64",
			source:      "flathub",
			description: "org.gimp.GIMP",
		},
		{
			// Only 2 fields (minimum)
			input:       "MyApp\tcom.example.MyApp",
			name:        "MyApp",
			version:     "",
			arch:        "",
			source:      "",
			description: "com.example.MyApp",
		},
		{
			// No name and no app ID
			input:   "\t\t1.0\tstable\tflathub\tx86_64",
			wantNil: true,
		},
		{
			// Empty line
			input:   "",
			wantNil: true,
		},
		{
			// Only one field
			input:   "onlyname",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := parseFlatpakLine(tt.input)
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
			if pkg.Source != tt.source {
				t.Errorf("source: got %q, want %q", pkg.Source, tt.source)
			}
			if pkg.PackageManager != "flatpak" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "flatpak")
			}
			if pkg.Description != tt.description {
				t.Errorf("description: got %q, want %q", pkg.Description, tt.description)
			}
		})
	}
}
