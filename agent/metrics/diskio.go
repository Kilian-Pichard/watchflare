package metrics

import (
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
)

// getDiskIOCounters returns cumulative disk read/write bytes across all real disks
func getDiskIOCounters() (uint64, uint64, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	var totalRead, totalWrite uint64
	for name, stat := range ioCounters {
		if isRealDisk(name) {
			totalRead += stat.ReadBytes
			totalWrite += stat.WriteBytes
		}
	}

	return totalRead, totalWrite, nil
}

// isRealDisk filters out virtual and partition devices
func isRealDisk(name string) bool {
	// Skip loop devices
	if strings.HasPrefix(name, "loop") {
		return false
	}
	// Skip device-mapper
	if strings.HasPrefix(name, "dm-") {
		return false
	}
	// Skip ram devices
	if strings.HasPrefix(name, "ram") {
		return false
	}
	return true
}
