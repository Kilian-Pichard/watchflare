package packages

import (
	"testing"
)

func TestParseYarnNameVersion(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
	}{
		{"typescript@5.3.3", "typescript", "5.3.3"},
		{"@angular/cli@17.1.0", "@angular/cli", "17.1.0"},
		{"yarn@1.22.21", "yarn", "1.22.21"},
		// No version
		{"packageonly", "packageonly", ""},
		// Scoped, no version
		{"@scope/pkg", "@scope/pkg", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version := parseYarnNameVersion(tt.input)
			if name != tt.name {
				t.Errorf("name: got %q, want %q", name, tt.name)
			}
			if version != tt.version {
				t.Errorf("version: got %q, want %q", version, tt.version)
			}
		})
	}
}

func TestParseYarnTreeNode(t *testing.T) {
	tests := []struct {
		name    string
		node    map[string]interface{}
		wantNil bool
		pkgName string
		pkgVer  string
	}{
		{
			name:    "valid node",
			node:    map[string]interface{}{"name": "typescript@5.3.3"},
			pkgName: "typescript",
			pkgVer:  "5.3.3",
		},
		{
			name:    "scoped package",
			node:    map[string]interface{}{"name": "@angular/cli@17.1.0"},
			pkgName: "@angular/cli",
			pkgVer:  "17.1.0",
		},
		{
			name:    "empty name",
			node:    map[string]interface{}{"name": ""},
			wantNil: true,
		},
		{
			name:    "missing name field",
			node:    map[string]interface{}{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := parseYarnTreeNode(tt.node)
			if tt.wantNil {
				if pkg != nil {
					t.Errorf("expected nil, got %+v", pkg)
				}
				return
			}
			if pkg == nil {
				t.Fatal("expected package, got nil")
			}
			if pkg.Name != tt.pkgName {
				t.Errorf("name: got %q, want %q", pkg.Name, tt.pkgName)
			}
			if pkg.Version != tt.pkgVer {
				t.Errorf("version: got %q, want %q", pkg.Version, tt.pkgVer)
			}
			if pkg.PackageManager != "yarn-global" {
				t.Errorf("package manager: got %q, want %q", pkg.PackageManager, "yarn-global")
			}
			if pkg.Source != "npmjs.com" {
				t.Errorf("source: got %q, want %q", pkg.Source, "npmjs.com")
			}
		})
	}
}
