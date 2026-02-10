package metrics

import (
	"log"
	"time"
	"watchflare-agent/sysinfo"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// Initialize initializes the metrics collector
// This is necessary on macOS where cpu.Percent() needs an initial baseline
func Initialize() {
	// Call cpu.Percent once to initialize internal state
	// This prevents getting 0% on first real measurement
	_, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Printf("Warning: Failed to initialize CPU metrics: %v", err)
	}
}

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

// Collect gathers system metrics based on environment configuration
// config parameter determines which metrics to collect (e.g., containers don't collect disk)
func Collect(config *sysinfo.MetricsConfig) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now().Unix(),
	}

	// CPU usage (averaged over 1 second)
	if config.CollectCPU {
		cpuPercent, err := cpu.Percent(time.Second, false)
		if err == nil && len(cpuPercent) > 0 {
			metrics.CPUUsagePercent = cpuPercent[0]
		}
	}

	// Memory stats
	if config.CollectMemory {
		memStats, err := mem.VirtualMemory()
		if err == nil {
			metrics.MemoryTotalBytes = memStats.Total
			metrics.MemoryUsedBytes = memStats.Used
			metrics.MemoryAvailableBytes = memStats.Available
		}
	}

	// Load average
	if config.CollectLoadAvg {
		loadStats, err := load.Avg()
		if err == nil {
			metrics.LoadAvg1Min = loadStats.Load1
			metrics.LoadAvg5Min = loadStats.Load5
			metrics.LoadAvg15Min = loadStats.Load15
		}
	}

	// Disk usage (root partition) - SKIPPED for containers to avoid double-counting
	if config.CollectDisk {
		diskStats, err := disk.Usage("/")
		if err == nil {
			metrics.DiskTotalBytes = diskStats.Total
			metrics.DiskUsedBytes = diskStats.Used
		}
	}

	// System uptime
	uptimeSeconds, err := host.Uptime()
	if err == nil {
		metrics.UptimeSeconds = uptimeSeconds
	}

	return metrics, nil
}
