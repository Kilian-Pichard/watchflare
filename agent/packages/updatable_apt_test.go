package packages

import "testing"

func TestParseAptUpgradableLine(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		wantName        string
		wantVersion     string
		wantSecurity    bool
		wantOK          bool
	}{
		{
			name:         "security update",
			line:         "curl/stable-security 8.11.1-1 amd64 [upgradable from: 8.11.0-6]",
			wantName:     "curl",
			wantVersion:  "8.11.1-1",
			wantSecurity: true,
			wantOK:       true,
		},
		{
			name:         "regular update",
			line:         "bash/stable 5.2.21-2 amd64 [upgradable from: 5.2.15-2]",
			wantName:     "bash",
			wantVersion:  "5.2.21-2",
			wantSecurity: false,
			wantOK:       true,
		},
		{
			name:         "ubuntu security repo",
			line:         "openssl/focal-security 3.3.2-2 amd64 [upgradable from: 3.3.1-2]",
			wantName:     "openssl",
			wantVersion:  "3.3.2-2",
			wantSecurity: true,
			wantOK:       true,
		},
		{
			name:    "listing header",
			line:    "Listing... Done",
			wantOK:  false,
		},
		{
			name:    "empty line",
			line:    "",
			wantOK:  false,
		},
		{
			name:    "no slash in first field",
			line:    "curl 8.11.1-1 amd64",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, status, ok := parseAptUpgradableLine(tt.line)
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
			if status.HasSecurityUpdate != tt.wantSecurity {
				t.Errorf("HasSecurityUpdate = %v, want %v", status.HasSecurityUpdate, tt.wantSecurity)
			}
		})
	}
}
