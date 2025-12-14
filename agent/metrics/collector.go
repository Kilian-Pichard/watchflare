package metrics

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemMetrics represents collected system metrics
type SystemMetrics struct {
	CPUUsagePercent      float64
	MemoryTotalBytes     uint64
	MemoryUsedBytes      uint64
	MemoryAvailableBytes uint64
	LoadAvg1Min          float64
	LoadAvg5Min          float64
	LoadAvg15Min         float64
	DiskTotalBytes       uint64
	DiskUsedBytes        uint64
	UptimeSeconds        uint64
	Timestamp            int64
}

// Collect gathers all system metrics
func Collect() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now().Unix(),
	}

	// CPU usage (averaged over 1 second)
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics.CPUUsagePercent = cpuPercent[0]
	}

	// Memory stats
	memStats, err := mem.VirtualMemory()
	if err == nil {
		metrics.MemoryTotalBytes = memStats.Total
		metrics.MemoryUsedBytes = memStats.Used
		metrics.MemoryAvailableBytes = memStats.Available
	}

	// Load average
	loadStats, err := load.Avg()
	if err == nil {
		metrics.LoadAvg1Min = loadStats.Load1
		metrics.LoadAvg5Min = loadStats.Load5
		metrics.LoadAvg15Min = loadStats.Load15
	}

	// Disk usage (root partition)
	diskStats, err := disk.Usage("/")
	if err == nil {
		metrics.DiskTotalBytes = diskStats.Total
		metrics.DiskUsedBytes = diskStats.Used
	}

	// System uptime
	uptimeSeconds, err := host.Uptime()
	if err == nil {
		metrics.UptimeSeconds = uptimeSeconds
	}

	return metrics, nil
}
