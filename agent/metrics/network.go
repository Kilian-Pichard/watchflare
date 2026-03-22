package metrics

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/net"
)

// getNetworkCounters returns cumulative network RX/TX bytes (aggregated across all interfaces)
func getNetworkCounters() (uint64, uint64, error) {
	ioCounters, err := net.IOCounters(false) // false = aggregated across all interfaces
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get network counters: %w", err)
	}
	if len(ioCounters) == 0 {
		return 0, 0, fmt.Errorf("no network counters available")
	}

	stat := ioCounters[0]
	return stat.BytesRecv, stat.BytesSent, nil
}
