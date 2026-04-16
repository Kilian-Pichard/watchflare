package packages

import (
	"testing"
)

func TestParsePnpmOutdated(t *testing.T) {
	input := []byte(`{
		"typescript": {"current": "5.9.3", "latest": "6.0.2", "wanted": "5.9.3", "isDeprecated": false, "dependencyType": "dependencies"},
		"prettier":   {"current": "3.8.3", "latest": "3.8.3", "wanted": "3.8.3", "isDeprecated": false, "dependencyType": "dependencies"}
	}`)

	updates, err := parsePnpmOutdated(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// prettier is up to date — should not appear
	if _, ok := updates["prettier"]; ok {
		t.Error("prettier should not be in updates (already up to date)")
	}

	u, ok := updates["typescript"]
	if !ok {
		t.Fatal("expected typescript in updates")
	}
	if u.AvailableVersion != "6.0.2" {
		t.Errorf("got %q, want %q", u.AvailableVersion, "6.0.2")
	}
	if u.HasSecurityUpdate {
		t.Error("HasSecurityUpdate should be false")
	}
}

func TestParsePnpmOutdated_Empty(t *testing.T) {
	updates, err := parsePnpmOutdated([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestParsePnpmOutdated_InvalidJSON(t *testing.T) {
	_, err := parsePnpmOutdated([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
