package packages

import (
	"testing"
)

func TestCLIToolsCollector_Name(t *testing.T) {
	collector := &CLIToolsCollector{}
	if collector.Name() != "cli-tools" {
		t.Errorf("expected name 'cli-tools', got '%s'", collector.Name())
	}
}

func TestCLIToolsCollector_IsAvailable(t *testing.T) {
	collector := &CLIToolsCollector{}
	if !collector.IsAvailable() {
		t.Error("CLIToolsCollector should always be available")
	}
}

func TestCLIToolsCollector_Collect(t *testing.T) {
	collector := &CLIToolsCollector{}
	packages, err := collector.Collect()

	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	for _, pkg := range packages {
		if pkg.Name == "" {
			t.Error("package name should not be empty")
		}
		if pkg.Version == "" {
			t.Error("package version should not be empty")
		}
		if pkg.PackageManager != "cli-tools" {
			t.Errorf("expected package_manager 'cli-tools', got '%s'", pkg.PackageManager)
		}
		t.Logf("found: %s v%s (category: %s, path: %s)", pkg.Name, pkg.Version, pkg.Source, pkg.Description)
	}
}

func TestParseVersion(t *testing.T) {
	collector := &CLIToolsCollector{}

	tests := []struct {
		input    string
		expected string
	}{
		{"Docker version 24.0.7, build afdd53b", "24.0.7"},
		{"git version 2.39.3", "2.39.3"},
		{"v1.28.0", "1.28.0"},
		{"node v20.10.0", "20.10.0"},
		{"Python 3.11.5", "3.11.5"},
		{"kubectl version 1.28.3", "1.28.3"},
		{"go version go1.21.4 darwin/arm64", "1.21.4"},
		{"terraform v1.6.4", "1.6.4"},
		{`helm version.BuildInfo{Version:"v3.13.2"`, "3.13.2"},
		{"cargo 1.74.0", "1.74.0"},
		// X.Y format (e.g. jq)
		{"jq-1.6", "1.6"},
		{"jq-1.7", "1.7"},
		{"jq-1.6\n", "1.6"},
		// Edge cases
		{"", ""},
		{"no version here", ""},
		{"only text without numbers", ""},
	}

	for _, tt := range tests {
		result := collector.parseVersion(tt.input)
		if result != tt.expected {
			t.Errorf("parseVersion(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
