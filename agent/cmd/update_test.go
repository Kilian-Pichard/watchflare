package cmd

import "testing"

func TestIsBrewPath_AppleSilicon(t *testing.T) {
	if !isBrewPath("/opt/homebrew/bin/watchflare-agent") {
		t.Error("expected true for Apple Silicon Homebrew path")
	}
}

func TestIsBrewPath_IntelCellar(t *testing.T) {
	if !isBrewPath("/usr/local/Cellar/watchflare-agent/1.2.0/bin/watchflare-agent") {
		t.Error("expected true for Intel Cellar path")
	}
}

func TestIsBrewPath_IntelSymlink(t *testing.T) {
	// /usr/local/bin symlink does not contain /homebrew/ or /Cellar/ — falls back to brew list
	if isBrewPath("/usr/local/bin/watchflare-agent") {
		t.Error("expected false for Intel symlink path")
	}
}

func TestIsBrewPath_Linux(t *testing.T) {
	if isBrewPath("/usr/local/bin/watchflare-agent") {
		t.Error("expected false for Linux path")
	}
}

func TestIsBrewPath_Empty(t *testing.T) {
	if isBrewPath("") {
		t.Error("expected false for empty path")
	}
}
