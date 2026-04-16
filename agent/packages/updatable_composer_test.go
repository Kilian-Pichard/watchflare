package packages

import (
	"testing"
)

func TestParseComposerOutdated(t *testing.T) {
	input := []byte(`{
		"installed": [
			{"name": "vendor/tool-a", "version": "1.0.0", "latest": "2.0.0"},
			{"name": "vendor/tool-b", "version": "3.1.0", "latest": "3.1.0"}
		]
	}`)

	updates, err := parseComposerOutdated(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// tool-b is up to date — should not appear
	if _, ok := updates["vendor/tool-b"]; ok {
		t.Error("vendor/tool-b should not be in updates (already up to date)")
	}

	u, ok := updates["vendor/tool-a"]
	if !ok {
		t.Fatal("expected vendor/tool-a in updates")
	}
	if u.AvailableVersion != "2.0.0" {
		t.Errorf("got %q, want %q", u.AvailableVersion, "2.0.0")
	}
	if u.HasSecurityUpdate {
		t.Error("HasSecurityUpdate should be false")
	}
}

func TestParseComposerOutdated_Empty(t *testing.T) {
	updates, err := parseComposerOutdated([]byte(`{"installed": []}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestParseComposerOutdated_InvalidJSON(t *testing.T) {
	_, err := parseComposerOutdated([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
