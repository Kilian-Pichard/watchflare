package packages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppNameFromInfo(t *testing.T) {
	tests := []struct {
		info    appInfo
		appPath string
		want    string
	}{
		{
			// Display name takes priority
			info:    appInfo{CFBundleDisplayName: "My App", CFBundleName: "MyApp"},
			appPath: "/Applications/MyApp.app",
			want:    "My App",
		},
		{
			// Falls back to bundle name
			info:    appInfo{CFBundleName: "MyApp"},
			appPath: "/Applications/MyApp.app",
			want:    "MyApp",
		},
		{
			// Falls back to filename
			info:    appInfo{},
			appPath: "/Applications/MyApp.app",
			want:    "MyApp",
		},
	}

	for _, tt := range tests {
		got := appNameFromInfo(tt.info, tt.appPath)
		if got != tt.want {
			t.Errorf("appNameFromInfo(%+v, %q) = %q, want %q", tt.info, tt.appPath, got, tt.want)
		}
	}
}

func TestAppVersionFromInfo(t *testing.T) {
	tests := []struct {
		info appInfo
		want string
	}{
		{
			// Short version takes priority
			info: appInfo{CFBundleShortVersionString: "1.2.3", CFBundleVersion: "100"},
			want: "1.2.3",
		},
		{
			// Falls back to bundle version
			info: appInfo{CFBundleVersion: "100"},
			want: "100",
		},
		{
			// Falls back to "unknown"
			info: appInfo{},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		got := appVersionFromInfo(tt.info)
		if got != tt.want {
			t.Errorf("appVersionFromInfo(%+v) = %q, want %q", tt.info, got, tt.want)
		}
	}
}

func TestDetectMacOSSource_Utilities(t *testing.T) {
	got := detectMacOSSource("/Applications/Utilities/Terminal.app")
	if got != "system" {
		t.Errorf("expected %q, got %q", "system", got)
	}
}

func TestDetectMacOSSource_SystemApp(t *testing.T) {
	got := detectMacOSSource("/Applications/Safari.app")
	if got != "system" {
		t.Errorf("expected %q, got %q", "system", got)
	}
}

func TestDetectMacOSSource_Manual(t *testing.T) {
	got := detectMacOSSource("/Applications/SomeThirdPartyApp.app")
	if got != "manual" {
		t.Errorf("expected %q, got %q", "manual", got)
	}
}

func TestDetectMacOSSource_AppStore(t *testing.T) {
	// Create a temp .app bundle with a MAS receipt
	dir := t.TempDir()
	appPath := filepath.Join(dir, "TestApp.app")
	receiptDir := filepath.Join(appPath, "Contents", "_MASReceipt")
	if err := os.MkdirAll(receiptDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(receiptDir, "receipt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	got := detectMacOSSource(appPath)
	if got != "app-store" {
		t.Errorf("expected %q, got %q", "app-store", got)
	}
}
