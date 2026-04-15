package packages

import "testing"

func TestParseDnfCheckUpdateLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantName    string
		wantVersion string
		wantOK      bool
	}{
		{
			name:        "x86_64 package",
			line:        "curl.x86_64   8.11.1-1.fc40   updates",
			wantName:    "curl",
			wantVersion: "8.11.1-1.fc40",
			wantOK:      true,
		},
		{
			name:        "noarch package",
			line:        "python3-pip.noarch   24.0-1.el9   baseos",
			wantName:    "python3-pip",
			wantVersion: "24.0-1.el9",
			wantOK:      true,
		},
		{
			// Epoch prefix is common on Fedora (DNF5 output)
			name:        "package with epoch",
			line:        "NetworkManager.aarch64   1:1.54.3-2.fc43   updates",
			wantName:    "NetworkManager",
			wantVersion: "1:1.54.3-2.fc43",
			wantOK:      true,
		},
		{
			name:   "empty line",
			line:   "",
			wantOK: false,
		},
		{
			name:   "only one field",
			line:   "curl.x86_64",
			wantOK: false,
		},
		// DNF5 header lines must be skipped (no dot in first field)
		{
			name:   "DNF5 header: Updating and loading repositories",
			line:   "Updating and loading repositories:",
			wantOK: false,
		},
		{
			name:   "DNF5 header: Repositories loaded",
			line:   "Repositories loaded.",
			wantOK: false,
		},
		{
			name:   "DNF4 section: Obsoleting packages",
			line:   "Obsoleting packages",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, version, ok := parseDnfCheckUpdateLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}
