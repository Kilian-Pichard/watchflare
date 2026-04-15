package packages

import "testing"

func TestParsePacmanUpdateLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantName    string
		wantVersion string
		wantOK      bool
	}{
		{
			name:        "simple package",
			line:        "curl 8.10.1-1 -> 8.11.1-1",
			wantName:    "curl",
			wantVersion: "8.11.1-1",
			wantOK:      true,
		},
		{
			name:        "package with hyphen in name",
			line:        "linux-headers 6.12.4.arch1-1 -> 6.12.6.arch1-1",
			wantName:    "linux-headers",
			wantVersion: "6.12.6.arch1-1",
			wantOK:      true,
		},
		{
			name:    "empty line",
			line:    "",
			wantOK:  false,
		},
		{
			name:    "missing arrow",
			line:    "curl 8.10.1-1 8.11.1-1",
			wantOK:  false,
		},
		{
			name:    "too few fields",
			line:    "curl 8.10.1-1",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, status, ok := parsePacmanUpdateLine(tt.line)
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
				t.Error("HasSecurityUpdate should always be false for Arch")
			}
		})
	}
}
