package packages

import (
	"strings"
	"testing"
)

func TestParsePipxVenv(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		venvData    map[string]interface{}
		wantName    string
		wantVersion string
		wantCmds    bool
	}{
		{
			name:    "full venv with commands",
			appName: "black",
			venvData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"main_package": map[string]interface{}{
						"package_version": "23.12.1",
						"apps":            []interface{}{"black", "blackd"},
					},
					"python_version": "3.11.5",
				},
			},
			wantName:    "black",
			wantVersion: "23.12.1",
			wantCmds:    true,
		},
		{
			name:    "venv without apps",
			appName: "mypy",
			venvData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"main_package": map[string]interface{}{
						"package_version": "1.8.0",
					},
					"python_version": "3.11.5",
				},
			},
			wantName:    "mypy",
			wantVersion: "1.8.0",
			wantCmds:    false,
		},
		{
			name:    "name from metadata when appName empty",
			appName: "",
			venvData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"main_package": map[string]interface{}{
						"package":         "ruff",
						"package_version": "0.1.14",
					},
					"python_version": "3.12.0",
				},
			},
			wantName:    "ruff",
			wantVersion: "0.1.14",
		},
		{
			name:    "no metadata",
			appName: "tool",
			venvData: map[string]interface{}{},
			wantName:    "tool",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := parsePipxVenv(tt.appName, tt.venvData)
			if pkg == nil {
				t.Fatal("expected package, got nil")
			}
			if pkg.Name != tt.wantName {
				t.Errorf("name: got %q, want %q", pkg.Name, tt.wantName)
			}
			if pkg.Version != tt.wantVersion {
				t.Errorf("version: got %q, want %q", pkg.Version, tt.wantVersion)
			}
			if pkg.PackageManager != "pipx" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "pipx")
			}
			if pkg.Source != "pypi.org" {
				t.Errorf("source: got %q, want %q", pkg.Source, "pypi.org")
			}
			if tt.wantCmds && !strings.Contains(pkg.Description, "Commands:") {
				t.Errorf("expected description to contain commands, got %q", pkg.Description)
			}
		})
	}
}

func TestParsePipxOutput(t *testing.T) {
	output := []byte(`{
		"venvs": {
			"black": {
				"metadata": {
					"main_package": {"package_version": "23.12.1"},
					"python_version": "3.11.5"
				},
				"apps": ["black", "blackd"]
			}
		}
	}`)

	pkgs, err := parsePipxOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "black" {
		t.Errorf("name: got %q, want %q", pkgs[0].Name, "black")
	}
}

func TestParsePipxOutput_NoVenvs(t *testing.T) {
	pkgs, err := parsePipxOutput([]byte(`{"venvs": {}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParsePipxOutput_InvalidJSON(t *testing.T) {
	_, err := parsePipxOutput([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
