//go:build darwin

package metrics

import (
	"fmt"
	"os/exec"

	"github.com/shirou/gopsutil/v4/disk"
	"howett.net/plist"
)

type diskutilInfo struct {
	APFSContainerSize uint64 `plist:"APFSContainerSize"`
	APFSContainerFree uint64 `plist:"APFSContainerFree"`
	FilesystemType    string `plist:"FilesystemType"`
}

func getDiskUsage() (total uint64, used uint64, err error) {
	// Try APFS-aware method first
	out, execErr := exec.Command("/usr/sbin/diskutil", "info", "-plist", "/").Output()
	if execErr == nil {
		var info diskutilInfo
		if _, plistErr := plist.Unmarshal(out, &info); plistErr == nil && info.FilesystemType == "apfs" && info.APFSContainerSize > 0 {
			used := uint64(0)
			if info.APFSContainerSize >= info.APFSContainerFree {
				used = info.APFSContainerSize - info.APFSContainerFree
			}
			return info.APFSContainerSize, used, nil
		}
	}

	// Fallback to gopsutil for non-APFS filesystems
	diskStats, err := disk.Usage("/")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get disk usage: %w", err)
	}
	return diskStats.Total, diskStats.Used, nil
}
