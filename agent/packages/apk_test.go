package packages

import "testing"

// --- splitNameVersion ---

func TestSplitNameVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"musl-1.2.4-r2", "musl", "1.2.4-r2"},
		{"alpine-base-3.18.4-r0", "alpine-base", "3.18.4-r0"},
		{"libgcc-13.2.1-r0", "libgcc", "13.2.1-r0"},
		{"lib2to3-1.0-r0", "lib2to3", "1.0-r0"},
		{"openssl-3.1.4-r1", "openssl", "3.1.4-r1"},
		{"nounversion", "nounversion", ""},
		{"noversion-only", "noversion-only", ""},
	}

	for _, tt := range tests {
		name, version := splitNameVersion(tt.input)
		if name != tt.wantName {
			t.Errorf("splitNameVersion(%q): name = %q, want %q", tt.input, name, tt.wantName)
		}
		if version != tt.wantVersion {
			t.Errorf("splitNameVersion(%q): version = %q, want %q", tt.input, version, tt.wantVersion)
		}
	}
}

// --- parseApkSize ---

func TestParseApkSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"4096", 4096},
		{"624 KiB", 624 * 1024},
		{"1.5 MiB", int64(1.5 * 1024 * 1024)},
		{"2 GiB", 2 * 1024 * 1024 * 1024},
		{"512 k", 512 * 1024},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := parseApkSize(tt.input)
		if got != tt.want {
			t.Errorf("parseApkSize(%q): got %d, want %d", tt.input, got, tt.want)
		}
	}
}

// --- parseBlock ---

func TestParseBlock_ValidBlock(t *testing.T) {
	block := []string{
		"musl-1.2.4-r2 description:",
		"A C library designed for Linux",
		"musl-1.2.4-r2 installed size:",
		"624 KiB",
		"musl-1.2.4-r2 arch:",
		"x86_64",
	}

	pkg := parseApkBlock(block)
	if pkg == nil {
		t.Fatal("expected non-nil package")
	}
	if pkg.Name != "musl" {
		t.Errorf("expected name %q, got %q", "musl", pkg.Name)
	}
	if pkg.Version != "1.2.4-r2" {
		t.Errorf("expected version %q, got %q", "1.2.4-r2", pkg.Version)
	}
	if pkg.Description != "A C library designed for Linux" {
		t.Errorf("unexpected description: %q", pkg.Description)
	}
	if pkg.PackageSize != 624*1024 {
		t.Errorf("expected size %d, got %d", 624*1024, pkg.PackageSize)
	}
	if pkg.Architecture != "x86_64" {
		t.Errorf("expected arch %q, got %q", "x86_64", pkg.Architecture)
	}
	if pkg.PackageManager != "apk" {
		t.Errorf("expected package manager %q, got %q", "apk", pkg.PackageManager)
	}
}

func TestParseBlock_Empty(t *testing.T) {
	if parseApkBlock(nil) != nil {
		t.Error("expected nil for empty block")
	}
	if parseApkBlock([]string{}) != nil {
		t.Error("expected nil for empty block")
	}
}

func TestParseBlock_NoSpace(t *testing.T) {
	if parseApkBlock([]string{"nospace"}) != nil {
		t.Error("expected nil when first line has no space")
	}
}
