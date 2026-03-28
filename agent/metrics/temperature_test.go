package metrics

import "testing"

func TestIsCPUSensor(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"coretemp Core 0", true},
		{"k10temp Tctl", true},
		{"cpu_thermal", true},
		{"Package id 0", true},
		{"Tctl", true},
		{"CPU Temp", true},
		{"CPU Die", true},
		{"PMU tdie1", true},
		{"fan0", false},
		{"battery", false},
		{"ambient", false},
	}

	for _, tt := range tests {
		got := isCPUSensor(tt.key)
		if got != tt.want {
			t.Errorf("isCPUSensor(%q): got %v, want %v", tt.key, got, tt.want)
		}
	}
}
