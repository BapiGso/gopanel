package monitor

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

func isDisplayableDiskPartition(partition disk.PartitionStat, usage *disk.UsageStat) bool {
	if partition.Mountpoint == "" || partition.Device == "" || partition.Device == "none" || usage == nil || usage.Total == 0 {
		return false
	}
	if hasMountOption(partition, "bind") {
		return false
	}
	return !strings.EqualFold(partition.Fstype, "overlay") || filepath.ToSlash(partition.Mountpoint) == "/"
}

func hasMountOption(partition disk.PartitionStat, option string) bool {
	return slices.Contains(partition.Opts, option)
}
