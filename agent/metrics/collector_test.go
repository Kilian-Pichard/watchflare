package metrics

import (
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	metrics, err := Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Verify timestamp is recent (within last 5 seconds)
	now := time.Now().Unix()
	if metrics.Timestamp < now-5 || metrics.Timestamp > now {
		t.Errorf("Timestamp = %v, want recent timestamp around %v", metrics.Timestamp, now)
	}

	// CPU usage should be between 0 and 100
	if metrics.CPUUsagePercent < 0 || metrics.CPUUsagePercent > 100 {
		t.Errorf("CPUUsagePercent = %v, want between 0 and 100", metrics.CPUUsagePercent)
	}

	// Memory total should be > 0
	if metrics.MemoryTotalBytes == 0 {
		t.Error("MemoryTotalBytes is 0")
	}

	// Memory used should be > 0 and <= total
	if metrics.MemoryUsedBytes == 0 {
		t.Error("MemoryUsedBytes is 0")
	}
	if metrics.MemoryUsedBytes > metrics.MemoryTotalBytes {
		t.Errorf("MemoryUsedBytes (%v) > MemoryTotalBytes (%v)",
			metrics.MemoryUsedBytes, metrics.MemoryTotalBytes)
	}

	// Memory available should be <= total
	if metrics.MemoryAvailableBytes > metrics.MemoryTotalBytes {
		t.Errorf("MemoryAvailableBytes (%v) > MemoryTotalBytes (%v)",
			metrics.MemoryAvailableBytes, metrics.MemoryTotalBytes)
	}

	// Load averages should be >= 0
	if metrics.LoadAvg1Min < 0 {
		t.Errorf("LoadAvg1Min = %v, want >= 0", metrics.LoadAvg1Min)
	}
	if metrics.LoadAvg5Min < 0 {
		t.Errorf("LoadAvg5Min = %v, want >= 0", metrics.LoadAvg5Min)
	}
	if metrics.LoadAvg15Min < 0 {
		t.Errorf("LoadAvg15Min = %v, want >= 0", metrics.LoadAvg15Min)
	}

	// Disk total should be > 0
	if metrics.DiskTotalBytes == 0 {
		t.Error("DiskTotalBytes is 0")
	}

	// Disk used should be > 0 and <= total
	if metrics.DiskUsedBytes == 0 {
		t.Error("DiskUsedBytes is 0")
	}
	if metrics.DiskUsedBytes > metrics.DiskTotalBytes {
		t.Errorf("DiskUsedBytes (%v) > DiskTotalBytes (%v)",
			metrics.DiskUsedBytes, metrics.DiskTotalBytes)
	}

	// Uptime should be > 0 (system has been running)
	if metrics.UptimeSeconds == 0 {
		t.Error("UptimeSeconds is 0")
	}

	// Log metrics for visibility
	t.Logf("CPU Usage: %.2f%%", metrics.CPUUsagePercent)
	t.Logf("Memory: %d MB / %d MB", metrics.MemoryUsedBytes/1024/1024, metrics.MemoryTotalBytes/1024/1024)
	t.Logf("Load Average: %.2f, %.2f, %.2f", metrics.LoadAvg1Min, metrics.LoadAvg5Min, metrics.LoadAvg15Min)
	t.Logf("Disk: %d GB / %d GB", metrics.DiskUsedBytes/1024/1024/1024, metrics.DiskTotalBytes/1024/1024/1024)
	t.Logf("Uptime: %d seconds", metrics.UptimeSeconds)
}
