package packages

import "testing"

func TestParseApkVersionLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantName    string
		wantVersion string
		wantOK      bool
	}{
		{
			name:        "simple package",
			line:        "curl-8.11.0-r0         < 8.11.1-r0",
			wantName:    "curl",
			wantVersion: "8.11.1-r0",
			wantOK:      true,
		},
		{
			name:        "package with hyphen in name",
			line:        "libssl3-3.3.2-r0       < 3.4.0-r0",
			wantName:    "libssl3",
			wantVersion: "3.4.0-r0",
			wantOK:      true,
		},
		{
			name:        "multi-hyphen name",
			line:        "py3-setuptools-68.0.0-r0 < 69.0.0-r0",
			wantName:    "py3-setuptools",
			wantVersion: "69.0.0-r0",
			wantOK:      true,
		},
		{
			name:    "header line",
			line:    "Installed:      Available:",
			wantOK:  false,
		},
		{
			name:    "empty line",
			line:    "",
			wantOK:  false,
		},
		{
			name:    "operator not <",
			line:    "curl-8.11.0-r0         = 8.11.0-r0",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, status, ok := parseApkVersionLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if status.AvailableVersion != tt.wantVersion {
				t.Errorf("AvailableVersion = %q, want %q", status.AvailableVersion, tt.wantVersion)
			}
			if status.HasSecurityUpdate {
				t.Error("HasSecurityUpdate should always be false for Alpine")
			}
		})
	}
}

func TestExtractApkPackageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"curl-8.11.0-r0", "curl"},
		{"libssl3-3.3.2-r0", "libssl3"},
		{"py3-setuptools-68.0.0-r0", "py3-setuptools"},
		{"busybox-1.36.1-r0", "busybox"},
		// No version → returns empty
		{"curl", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractApkPackageName(tt.input)
			if got != tt.want {
				t.Errorf("extractApkPackageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
