package packages

import (
	"testing"
)

func TestParsePipOutdated(t *testing.T) {
	input := []byte(`[
		{"name": "requests", "version": "2.28.0", "latest_version": "2.32.0", "latest_filetype": "wheel"},
		{"name": "setuptools", "version": "65.0.0", "latest_version": "75.0.0", "latest_filetype": "wheel"},
		{"name": "pip", "version": "23.0", "latest_version": "23.0", "latest_filetype": "wheel"}
	]`)

	updates, err := parsePipOutdated(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// pip is up to date — should not appear
	if _, ok := updates["pip"]; ok {
		t.Error("pip should not be in updates (already up to date)")
	}

	tests := []struct {
		name    string
		version string
	}{
		{"requests", "2.32.0"},
		{"setuptools", "75.0.0"},
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

func TestParsePipOutdated_Empty(t *testing.T) {
	updates, err := parsePipOutdated([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestParsePipOutdated_InvalidJSON(t *testing.T) {
	_, err := parsePipOutdated([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
