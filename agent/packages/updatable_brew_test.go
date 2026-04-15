package packages

import "testing"

func TestParseBrewOutdatedJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]UpdateStatus
		wantErr bool
	}{
		{
			name: "formulae and casks",
			input: `{
				"formulae": [
					{"name": "curl", "current_version": "8.11.1"},
					{"name": "git", "current_version": "2.44.0"}
				],
				"casks": [
					{"name": "firefox", "current_version": "123.0"}
				]
			}`,
			want: map[string]UpdateStatus{
				"curl":    {AvailableVersion: "8.11.1", HasSecurityUpdate: false},
				"git":     {AvailableVersion: "2.44.0", HasSecurityUpdate: false},
				"firefox": {AvailableVersion: "123.0", HasSecurityUpdate: false},
			},
		},
		{
			name: "formulae only",
			input: `{
				"formulae": [{"name": "openssl", "current_version": "3.3.2"}],
				"casks": []
			}`,
			want: map[string]UpdateStatus{
				"openssl": {AvailableVersion: "3.3.2", HasSecurityUpdate: false},
			},
		},
		{
			name:  "empty",
			input: `{"formulae": [], "casks": []}`,
			want:  map[string]UpdateStatus{},
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBrewOutdatedJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(tt.want))
			}
			for name, wantStatus := range tt.want {
				gotStatus, ok := got[name]
				if !ok {
					t.Errorf("missing entry for %q", name)
					continue
				}
				if gotStatus.AvailableVersion != wantStatus.AvailableVersion {
					t.Errorf("%q: AvailableVersion = %q, want %q", name, gotStatus.AvailableVersion, wantStatus.AvailableVersion)
				}
				if gotStatus.HasSecurityUpdate {
					t.Errorf("%q: HasSecurityUpdate should always be false for brew", name)
				}
			}
		})
	}
}
