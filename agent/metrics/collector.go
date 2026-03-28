package metrics

import (
	"log/slog"
	"sync"
	"time"
	"watchflare-agent/sysinfo"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

// Initialize initializes the metrics collector
// This is necessary on macOS where cpu.Percent() needs an initial baseline
func Initialize() {
	// Call cpu.Percent once to initialize internal state
	// This prevents getting 0% on first real measurement
	_, err := cpu.Percent(time.Second, false)
	if err != nil {
		slog.Warn("failed to initialize CPU metrics", "error", err)
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

	// Disk I/O rates (bytes per second)
	DiskReadBytesPerSec  uint64
	DiskWriteBytesPerSec uint64

	// Network rates (bytes per second)
	NetworkRxBytesPerSec uint64
	NetworkTxBytesPerSec uint64

	// Temperature (physical servers only)
	CPUTemperatureCelsius float64

	// All sensor readings (temperature sensors, battery, storage, etc.)
	SensorReadings []SensorReading

	// Docker container metrics (only for hosts with containers)
	ContainerMetrics []ContainerMetric
}

// Package-level delta tracker for rate-based metrics (disk I/O, network)
var deltaTracker = NewDeltaTracker()

// dockerErrorLogged suppresses repeated Docker error logs after the first failure.
// Protected by dockerErrorMu.
var (
	dockerErrorLogged bool
	dockerErrorMu     sync.Mutex
)

// Collect gathers system metrics based on environment configuration
// config parameter determines which metrics to collect (e.g., containers don't collect disk)
func Collect(config *sysinfo.MetricsConfig) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now().Unix(),
	}

	// CPU usage (averaged over 1 second)
	if config.CollectCPU {
		cpuPercent, err := cpu.Percent(time.Second, false)
		if err != nil {
			slog.Debug("failed to collect CPU metrics", "error", err)
		} else if len(cpuPercent) > 0 {
			metrics.CPUUsagePercent = cpuPercent[0]
		}
	}

	// Memory stats
	if config.CollectMemory {
		memStats, err := mem.VirtualMemory()
		if err != nil {
			slog.Debug("failed to collect memory metrics", "error", err)
		} else {
			metrics.MemoryTotalBytes = memStats.Total
			metrics.MemoryUsedBytes = memStats.Total - memStats.Available
			metrics.MemoryAvailableBytes = memStats.Available
		}
	}

	// Load average
	if config.CollectLoadAvg {
		loadStats, err := load.Avg()
		if err != nil {
			slog.Debug("failed to collect load average", "error", err)
		} else {
			metrics.LoadAvg1Min = loadStats.Load1
			metrics.LoadAvg5Min = loadStats.Load5
			metrics.LoadAvg15Min = loadStats.Load15
		}
	}

	// Disk usage - SKIPPED for containers to avoid double-counting
	// On macOS: uses diskutil for APFS-accurate values (container level)
	// On Linux: uses gopsutil disk.Usage("/")
	if config.CollectDisk {
		total, used, diskErr := getDiskUsage()
		if diskErr != nil {
			slog.Debug("failed to collect disk usage", "error", diskErr)
		} else {
			metrics.DiskTotalBytes = total
			metrics.DiskUsedBytes = used
		}
	}

	// Disk I/O
	if config.CollectDiskIO {
		readBytes, writeBytes, ioErr := getDiskIOCounters()
		if ioErr != nil {
			slog.Debug("failed to collect disk I/O", "error", ioErr)
		} else {
			now := time.Now()
			metrics.DiskReadBytesPerSec, metrics.DiskWriteBytesPerSec = deltaTracker.ComputeDiskIORate(readBytes, writeBytes, now)
		}
	}

	// Network bandwidth
	if config.CollectNetwork {
		rxBytes, txBytes, netErr := getNetworkCounters()
		if netErr != nil {
			slog.Debug("failed to collect network counters", "error", netErr)
		} else {
			now := time.Now()
			metrics.NetworkRxBytesPerSec, metrics.NetworkTxBytesPerSec = deltaTracker.ComputeNetworkRate(rxBytes, txBytes, now)
		}
	}

	// Temperature (physical servers only) — single syscall for CPU temp + all readings
	if config.CollectTemperature {
		cpuTemp, readings, tempErr := collectTemperatures()
		if tempErr != nil {
			slog.Debug("failed to collect temperature sensors", "error", tempErr)
		} else {
			metrics.CPUTemperatureCelsius = cpuTemp
			metrics.SensorReadings = readings
		}
	}

	// Docker container metrics
	if config.CollectDockerCPU || config.CollectDockerMemory || config.CollectDockerNetwork {
		containerMetrics, containerErr := CollectContainerMetrics(deltaTracker)
		dockerErrorMu.Lock()
		if containerErr != nil {
			if !dockerErrorLogged {
				slog.Warn("failed to collect container metrics", "error", containerErr)
				dockerErrorLogged = true
			}
		} else if containerMetrics != nil {
			if dockerErrorLogged {
				slog.Info("container metrics collection recovered after previous error")
				dockerErrorLogged = false
			}
			metrics.ContainerMetrics = containerMetrics
		}
		dockerErrorMu.Unlock()
	}

	// System uptime
	uptimeSeconds, err := host.Uptime()
	if err == nil {
		metrics.UptimeSeconds = uptimeSeconds
	}

	return metrics, nil
}
