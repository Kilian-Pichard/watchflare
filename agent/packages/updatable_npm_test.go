package packages

import (
	"testing"
)

func TestParseNpmOutdated(t *testing.T) {
	input := []byte(`{
		"typescript": {"current": "6.0.2", "wanted": "6.0.2", "latest": "7.0.0"},
		"prettier":   {"current": "3.8.3", "wanted": "3.8.3", "latest": "3.8.3"},
		"ts-node":    {"current": "10.9.2", "wanted": "10.9.2", "latest": "11.0.0"}
	}`)

	updates, err := parseNpmOutdated(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// prettier is up to date — should not appear
	if _, ok := updates["prettier"]; ok {
		t.Error("prettier should not be in updates (already up to date)")
	}

	tests := []struct {
		name    string
		version string
	}{
		{"typescript", "7.0.0"},
		{"ts-node", "11.0.0"},
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

func TestParseNpmOutdated_Empty(t *testing.T) {
	updates, err := parseNpmOutdated([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestParseNpmOutdated_InvalidJSON(t *testing.T) {
	_, err := parseNpmOutdated([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
