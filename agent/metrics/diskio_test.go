package metrics

import "testing"

func TestIsRealDisk(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// Real disks
		{"sda", true},
		{"sdb", true},
		{"nvme0n1", true},
		{"vda", true},
		// Loop devices
		{"loop0", false},
		{"loop10", false},
		// Device-mapper
		{"dm-0", false},
		{"dm-1", false},
		// RAM devices
		{"ram0", false},
		{"ram10", false},
		// Compressed swap
		{"zram0", false},
		{"zram1", false},
		// Optical drives
		{"sr0", false},
		{"sr1", false},
	}

	for _, tt := range tests {
		got := isRealDisk(tt.name)
		if got != tt.want {
			t.Errorf("isRealDisk(%q): got %v, want %v", tt.name, got, tt.want)
		}
	}
}
