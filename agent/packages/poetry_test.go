package packages

import (
	"testing"
)

func TestParsePipFreezeLine(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
		ok      bool
	}{
		{"requests==2.31.0", "requests", "2.31.0", true},
		{"Django==4.2.9", "Django", "4.2.9", true},
		{"numpy==1.26.3", "numpy", "1.26.3", true},
		// Lines without == (e.g. editable installs "-e git+...") — skip
		{"-e git+https://github.com/org/pkg@abc#egg=pkg", "", "", false},
		{"", "", "", false},
		// Only package name, no version
		{"onlyname", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version, ok := parsePipFreezeLine(tt.input)
			if ok != tt.ok {
				t.Errorf("ok: got %v, want %v", ok, tt.ok)
				return
			}
			if !ok {
				return
			}
			if name != tt.name {
				t.Errorf("name: got %q, want %q", name, tt.name)
			}
			if version != tt.version {
				t.Errorf("version: got %q, want %q", version, tt.version)
			}
		})
	}
}
