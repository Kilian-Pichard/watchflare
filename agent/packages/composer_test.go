package packages

import (
	"fmt"
	"strings"
	"testing"
)

func TestComposerCollect_ParsesOutput(t *testing.T) {
	output := []byte(`{
		"installed": [
			{"name": "phpunit/phpunit", "version": "10.5.0", "description": "The PHP Unit Testing framework"},
			{"name": "symfony/console", "version": "6.4.0", "description": "Eases the creation of beautiful and testable command line interfaces"}
		]
	}`)

	pkgs, err := parseComposerJSON(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	tests := []struct{ name, version string }{
		{"phpunit/phpunit", "10.5.0"},
		{"symfony/console", "6.4.0"},
	}

	for i, tt := range tests {
		if pkgs[i].Name != tt.name {
			t.Errorf("[%d] name: got %q, want %q", i, pkgs[i].Name, tt.name)
		}
		if pkgs[i].Version != tt.version {
			t.Errorf("[%d] version: got %q, want %q", i, pkgs[i].Version, tt.version)
		}
		if pkgs[i].PackageManager != "composer" {
			t.Errorf("[%d] package manager: got %q, want %q", i, pkgs[i].PackageManager, "composer")
		}
		if pkgs[i].Source != "packagist.org" {
			t.Errorf("[%d] source: got %q, want %q", i, pkgs[i].Source, "packagist.org")
		}
	}
}

func TestComposerCollect_EmptyInstalled(t *testing.T) {
	output := []byte(`{"installed": []}`)

	pkgs, err := parseComposerJSON(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestComposerCollect_InvalidJSON(t *testing.T) {
	_, err := parseComposerJSON([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestComposerCollect_TruncatesDescription(t *testing.T) {
	long := strings.Repeat("a", 200)
	output := []byte(fmt.Sprintf(`{"installed": [{"name": "vendor/pkg", "version": "1.0.0", "description": "%s"}]}`, long))

	pkgs, err := parseComposerJSON(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if len([]rune(pkgs[0].Description)) > 100 {
		t.Errorf("description not truncated: len=%d", len([]rune(pkgs[0].Description)))
	}
}
