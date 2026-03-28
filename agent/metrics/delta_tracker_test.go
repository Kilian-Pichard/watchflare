package metrics

import (
	"testing"
	"time"
)

// --- ComputeDiskIORate ---

func TestDeltaTracker_DiskIO_FirstCallReturnsZero(t *testing.T) {
	dt := NewDeltaTracker()
	now := time.Now()

	r, w := dt.ComputeDiskIORate(1000, 500, now)
	if r != 0 || w != 0 {
		t.Errorf("first call: expected (0, 0), got (%d, %d)", r, w)
	}
}

func TestDeltaTracker_DiskIO_ComputesRate(t *testing.T) {
	dt := NewDeltaTracker()
	t0 := time.Now()
	t1 := t0.Add(2 * time.Second)

	dt.ComputeDiskIORate(0, 0, t0) // baseline

	r, w := dt.ComputeDiskIORate(2000, 4000, t1)

	// 2000 bytes / 2s = 1000 B/s, 4000 / 2s = 2000 B/s
	if r != 1000 {
		t.Errorf("expected read rate 1000, got %d", r)
	}
	if w != 2000 {
		t.Errorf("expected write rate 2000, got %d", w)
	}
}

func TestDeltaTracker_DiskIO_CounterRollover(t *testing.T) {
	dt := NewDeltaTracker()
	t0 := time.Now()
	t1 := t0.Add(time.Second)

	dt.ComputeDiskIORate(1000, 1000, t0)

	// Current < prev: rollover — rate should be 0
	r, w := dt.ComputeDiskIORate(500, 500, t1)
	if r != 0 || w != 0 {
		t.Errorf("rollover: expected (0, 0), got (%d, %d)", r, w)
	}
}

func TestDeltaTracker_DiskIO_ZeroElapsed(t *testing.T) {
	dt := NewDeltaTracker()
	now := time.Now()

	dt.ComputeDiskIORate(0, 0, now)

	// Same timestamp: elapsed = 0, must not divide by zero
	r, w := dt.ComputeDiskIORate(1000, 1000, now)
	if r != 0 || w != 0 {
		t.Errorf("zero elapsed: expected (0, 0), got (%d, %d)", r, w)
	}
}

// --- ComputeNetworkRate ---

func TestDeltaTracker_Network_FirstCallReturnsZero(t *testing.T) {
	dt := NewDeltaTracker()
	now := time.Now()

	rx, tx := dt.ComputeNetworkRate(5000, 2000, now)
	if rx != 0 || tx != 0 {
		t.Errorf("first call: expected (0, 0), got (%d, %d)", rx, tx)
	}
}

func TestDeltaTracker_Network_ComputesRate(t *testing.T) {
	dt := NewDeltaTracker()
	t0 := time.Now()
	t1 := t0.Add(4 * time.Second)

	dt.ComputeNetworkRate(0, 0, t0)

	rx, tx := dt.ComputeNetworkRate(8000, 4000, t1)

	// 8000/4s = 2000 B/s, 4000/4s = 1000 B/s
	if rx != 2000 {
		t.Errorf("expected rx rate 2000, got %d", rx)
	}
	if tx != 1000 {
		t.Errorf("expected tx rate 1000, got %d", tx)
	}
}

func TestDeltaTracker_Network_Rollover(t *testing.T) {
	dt := NewDeltaTracker()
	t0 := time.Now()
	t1 := t0.Add(time.Second)

	dt.ComputeNetworkRate(5000, 5000, t0)

	rx, tx := dt.ComputeNetworkRate(100, 100, t1)
	if rx != 0 || tx != 0 {
		t.Errorf("rollover: expected (0, 0), got (%d, %d)", rx, tx)
	}
}

// --- ComputeContainerNetworkRate ---

func TestDeltaTracker_ContainerNetwork_FirstCallReturnsZero(t *testing.T) {
	dt := NewDeltaTracker()
	now := time.Now()

	rx, tx := dt.ComputeContainerNetworkRate("abc123", 1000, 500, now)
	if rx != 0 || tx != 0 {
		t.Errorf("first call: expected (0, 0), got (%d, %d)", rx, tx)
	}
}

func TestDeltaTracker_ContainerNetwork_ComputesRate(t *testing.T) {
	dt := NewDeltaTracker()
	t0 := time.Now()
	t1 := t0.Add(2 * time.Second)

	dt.ComputeContainerNetworkRate("abc123", 0, 0, t0)

	rx, tx := dt.ComputeContainerNetworkRate("abc123", 6000, 2000, t1)

	if rx != 3000 {
		t.Errorf("expected rx 3000, got %d", rx)
	}
	if tx != 1000 {
		t.Errorf("expected tx 1000, got %d", tx)
	}
}

func TestDeltaTracker_ContainerNetwork_IndependentPerContainer(t *testing.T) {
	dt := NewDeltaTracker()
	t0 := time.Now()
	t1 := t0.Add(time.Second)

	dt.ComputeContainerNetworkRate("c1", 0, 0, t0)
	dt.ComputeContainerNetworkRate("c2", 0, 0, t0)

	rx1, _ := dt.ComputeContainerNetworkRate("c1", 1000, 0, t1)
	rx2, _ := dt.ComputeContainerNetworkRate("c2", 3000, 0, t1)

	if rx1 != 1000 {
		t.Errorf("c1: expected rx 1000, got %d", rx1)
	}
	if rx2 != 3000 {
		t.Errorf("c2: expected rx 3000, got %d", rx2)
	}
}

// --- computeCPUPercent ---

func TestComputeCPUPercent_Normal(t *testing.T) {
	stats := &dockerStatsResponse{}
	stats.CPUStats.CPUUsage.TotalUsage = 200_000_000
	stats.PreCPUStats.CPUUsage.TotalUsage = 100_000_000
	stats.CPUStats.SystemCPUUsage = 1_000_000_000
	stats.PreCPUStats.SystemCPUUsage = 900_000_000
	stats.CPUStats.OnlineCPUs = 2

	// cpuDelta=100M, systemDelta=100M, numCPUs=2 → 200%
	got := computeCPUPercent(stats)
	if got != 200.0 {
		t.Errorf("expected 200.0%%, got %f", got)
	}
}

func TestComputeCPUPercent_ZeroSystem(t *testing.T) {
	stats := &dockerStatsResponse{}
	stats.CPUStats.CPUUsage.TotalUsage = 100
	stats.PreCPUStats.CPUUsage.TotalUsage = 0
	stats.CPUStats.SystemCPUUsage = 0
	stats.PreCPUStats.SystemCPUUsage = 0

	got := computeCPUPercent(stats)
	if got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestComputeCPUPercent_Underflow(t *testing.T) {
	stats := &dockerStatsResponse{}
	// Current < previous → underflow guard
	stats.CPUStats.CPUUsage.TotalUsage = 50
	stats.PreCPUStats.CPUUsage.TotalUsage = 100
	stats.CPUStats.SystemCPUUsage = 1000
	stats.PreCPUStats.SystemCPUUsage = 900

	got := computeCPUPercent(stats)
	if got != 0 {
		t.Errorf("expected 0 on underflow, got %f", got)
	}
}

func TestComputeCPUPercent_ZeroOnlineCPUs_DefaultsToOne(t *testing.T) {
	stats := &dockerStatsResponse{}
	stats.CPUStats.CPUUsage.TotalUsage = 100_000_000
	stats.PreCPUStats.CPUUsage.TotalUsage = 0
	stats.CPUStats.SystemCPUUsage = 1_000_000_000
	stats.PreCPUStats.SystemCPUUsage = 0
	stats.CPUStats.OnlineCPUs = 0 // defaults to 1

	got := computeCPUPercent(stats)
	expected := (100_000_000.0 / 1_000_000_000.0) * 1 * 100.0
	if got != expected {
		t.Errorf("expected %f, got %f", expected, got)
	}
}

// --- truncateID ---

func TestTruncateID_LongID(t *testing.T) {
	id := "abcdef1234567890"
	got := truncateID(id)
	if got != "abcdef123456" {
		t.Errorf("expected 12-char prefix, got %q", got)
	}
}

func TestTruncateID_ShortID(t *testing.T) {
	id := "abc"
	got := truncateID(id)
	if got != "abc" {
		t.Errorf("expected unchanged short id, got %q", got)
	}
}

// --- isRealDisk ---

func TestIsRealDisk(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"sda", true},
		{"nvme0n1", true},
		{"loop0", false},
		{"loop10", false},
		{"dm-0", false},
		{"dm-1", false},
		{"ram0", false},
		{"ram10", false},
	}

	for _, tt := range tests {
		got := isRealDisk(tt.name)
		if got != tt.want {
			t.Errorf("isRealDisk(%q): got %v, want %v", tt.name, got, tt.want)
		}
	}
}

// --- isCPUSensor ---

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
