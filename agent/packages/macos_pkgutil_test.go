package packages

import (
	"testing"
	"time"
)

func TestExtractPkgName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"com.apple.pkg.Safari", "Safari"},
		{"com.microsoft.OneDrive", "OneDrive"},
		{"org.example", "example"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		got := extractPkgName(tt.input)
		if got != tt.want {
			t.Errorf("extractPkgName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectPkgSource(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"com.apple.pkg.Safari", "apple"},
		{"com.oracle.jdk", "oracle"},
		{"com.adobe.acrobat", "adobe"},
		{"com.microsoft.OneDrive", "microsoft"},
		{"com.google.Chrome", "google"},
		{"com.docker.docker", "docker"},
		{"io.tailscale.ipn.macos", "third-party"},
		{"org.example.app", "third-party"},
	}

	for _, tt := range tests {
		got := detectPkgSource(tt.input)
		if got != tt.want {
			t.Errorf("detectPkgSource(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParsePkgInfo(t *testing.T) {
	output := []byte(`package-id: com.apple.pkg.Safari
version: 17.0
volume: /
location: Applications
install-time: 1700000000
`)
	version, installedAt := parsePkgInfo(output)

	if version != "17.0" {
		t.Errorf("version: got %q, want %q", version, "17.0")
	}
	want := time.Unix(1700000000, 0)
	if !installedAt.Equal(want) {
		t.Errorf("installedAt: got %v, want %v", installedAt, want)
	}
}

func TestParsePkgInfo_MissingFields(t *testing.T) {
	output := []byte(`package-id: com.example.pkg
`)
	version, installedAt := parsePkgInfo(output)

	if version != "" {
		t.Errorf("expected empty version, got %q", version)
	}
	if !installedAt.IsZero() {
		t.Errorf("expected zero installedAt, got %v", installedAt)
	}
}

func TestParsePkgInfo_InvalidTimestamp(t *testing.T) {
	output := []byte(`version: 1.0
install-time: notanumber
`)
	_, installedAt := parsePkgInfo(output)

	if !installedAt.IsZero() {
		t.Errorf("expected zero installedAt for invalid timestamp, got %v", installedAt)
	}
}
