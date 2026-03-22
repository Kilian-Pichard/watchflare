// +build !darwin

package metrics

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"
)

func getDiskUsage() (total uint64, used uint64, err error) {
	diskStats, err := disk.Usage("/")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get disk usage: %w", err)
	}
	return diskStats.Total, diskStats.Used, nil
}
