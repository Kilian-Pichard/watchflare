package packages

import (
	"testing"
)

// --- isAppImage ---

func TestIsAppImage(t *testing.T) {
	a := &AppImageCollector{}
	tests := []struct {
		path string
		want bool
	}{
		{"/home/user/Apps/MyApp-1.2.3-x86_64.AppImage", true},
		{"/home/user/Apps/MyApp.appimage", true},
		{"/home/user/Apps/MyApp.APPIMAGE", true},
		{"/home/user/Apps/MyApp.AppImage", true},
		{"/home/user/Apps/notanappimage.exe", false},
		{"/home/user/Apps/noextension", false},
		{"", false},
	}

	for _, tt := range tests {
		got := a.isAppImage(tt.path)
		if got != tt.want {
			t.Errorf("isAppImage(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// --- parseAppImageName ---

func TestParseAppImageName(t *testing.T) {
	a := &AppImageCollector{}
	tests := []struct {
		filename    string
		wantName    string
		wantVersion string
	}{
		// Standard pattern: Name-version-arch.AppImage
		{"AppName-1.2.3-x86_64.AppImage", "AppName", "1.2.3"},
		// No arch: Name-version.AppImage
		{"MyTool-2.0.0.AppImage", "MyTool", "2.0.0"},
		// Lowercase extension
		{"firefox-121.0.appimage", "firefox", "121.0"},
		// All-caps extension
		{"tool-1.0.APPIMAGE", "tool", "1.0"},
		// Underscore separator: first digit-starting part is the version boundary
		{"App_Name_1.0.0.AppImage", "App-Name", "1.0.0"},
		// No version
		{"MyApp.AppImage", "MyApp", ""},
		// Arch only (stripped, leaving name only)
		{"MyApp-x86_64.AppImage", "MyApp", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			name, version := a.parseAppImageName(tt.filename)
			if name != tt.wantName {
				t.Errorf("name: got %q, want %q", name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("version: got %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

// --- detectArch ---

func TestDetectArch(t *testing.T) {
	a := &AppImageCollector{}
	tests := []struct {
		filename string
		want     string
	}{
		{"MyApp-1.0-x86_64.AppImage", "x86_64"},
		{"MyApp-1.0-amd64.AppImage", "x86_64"},
		{"MyApp-1.0-i686.AppImage", "i686"},
		{"MyApp-1.0-i386.AppImage", "i686"},
		{"MyApp-1.0-arm64.AppImage", "arm64"},
		{"MyApp-1.0-aarch64.AppImage", "arm64"},
		{"MyApp-1.0-armhf.AppImage", "armhf"},
		{"MyApp-1.0-armv7.AppImage", "armhf"},
		{"MyApp-1.0.AppImage", ""},
	}

	for _, tt := range tests {
		got := a.detectArch(tt.filename)
		if got != tt.want {
			t.Errorf("detectArch(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}
