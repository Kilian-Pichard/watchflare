package packages

import (
	"testing"
)

func TestParsePacmanSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1.23 MiB", 1289748},        // int64(1.23 * 1024 * 1024)
		{"456.78 KiB", 467742},       // int64(456.78 * 1024)
		{"2.00 GiB", 2147483648},     // 2 * 1024^3
		{"512 B", 512},
		{"", 0},
		{"invalid", 0},
		{"1.5 TB", 0}, // unknown unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parsePacmanSize(tt.input)
			if got != tt.want {
				t.Errorf("parsePacmanSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParsePacmanOutput(t *testing.T) {
	output := []byte(`Name            : firefox
Version         : 121.0-1
Description     : Standalone web browser from mozilla.org
Architecture    : x86_64
Repository      : extra
Install Date    : Mon 01 Jan 2024 12:00:00 PM UTC
Installed Size  : 256.00 MiB

Name            : git
Version         : 2.43.0-1
Description     : the fast distributed version control system
Architecture    : x86_64
Repository      : extra
Install Date    : Mon 15 Jan 2024 08:30:00 AM UTC
Installed Size  : 32.50 MiB

`)

	pkgs := parsePacmanOutput(output)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	firefox := pkgs[0]
	if firefox.Name != "firefox" {
		t.Errorf("name: got %q, want %q", firefox.Name, "firefox")
	}
	if firefox.Version != "121.0-1" {
		t.Errorf("version: got %q, want %q", firefox.Version, "121.0-1")
	}
	if firefox.Architecture != "x86_64" {
		t.Errorf("architecture: got %q, want %q", firefox.Architecture, "x86_64")
	}
	if firefox.Source != "extra" {
		t.Errorf("source: got %q, want %q", firefox.Source, "extra")
	}
	if firefox.PackageManager != "pacman" {
		t.Errorf("package manager: got %q, want %q", firefox.PackageManager, "pacman")
	}
	if firefox.PackageSize == 0 {
		t.Error("expected non-zero package size")
	}
	if firefox.InstalledAt.IsZero() {
		t.Error("expected non-zero install date")
	}
}

func TestParsePacmanOutput_Empty(t *testing.T) {
	pkgs := parsePacmanOutput([]byte(""))
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParsePacmanOutput_MissingNameOrVersion(t *testing.T) {
	// Block with no Name — should be skipped
	output := []byte(`Version         : 1.0.0
Architecture    : x86_64

`)
	pkgs := parsePacmanOutput(output)
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParsePacmanOutput_InvalidInstallDate(t *testing.T) {
	output := []byte(`Name            : mypkg
Version         : 1.0.0
Install Date    : not-a-date

`)
	pkgs := parsePacmanOutput(output)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if !pkgs[0].InstalledAt.IsZero() {
		t.Error("expected zero install date for invalid date string")
	}
}
