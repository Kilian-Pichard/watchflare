package packages

import (
	"testing"
)

func TestParseGemOutdated(t *testing.T) {
	input := []byte(`bundler (2.3.26 < 2.6.8)
rake (13.0.6 < 13.2.1)
rubygems-update (3.4.10 < 3.6.9)
`)

	updates := parseGemOutdated(input)

	tests := []struct {
		name    string
		version string
	}{
		{"bundler", "2.6.8"},
		{"rake", "13.2.1"},
		{"rubygems-update", "3.6.9"},
	}
	for _, tt := range tests {
		u, ok := updates[tt.name]
		if !ok {
			t.Errorf("expected %q in updates", tt.name)
			continue
		}
		if u.AvailableVersion != tt.version {
			t.Errorf("%s: got %q, want %q", tt.name, u.AvailableVersion, tt.version)
		}
		if u.HasSecurityUpdate {
			t.Errorf("%s: HasSecurityUpdate should be false", tt.name)
		}
	}
}

func TestParseGemOutdated_Empty(t *testing.T) {
	updates := parseGemOutdated([]byte(``))
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestParseGemOutdated_SkipsInvalidLines(t *testing.T) {
	input := []byte(`bundler (2.3.26 < 2.6.8)
not a valid line
rake (13.0.6 < 13.2.1)
`)
	updates := parseGemOutdated(input)
	if len(updates) != 2 {
		t.Errorf("expected 2 updates, got %d", len(updates))
	}
}
