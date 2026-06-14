package monitor

import (
	"testing"

	"github.com/shirou/gopsutil/v4/disk"
)

func TestIsDisplayableDiskPartition(t *testing.T) {
	tests := []struct {
		name      string
		partition disk.PartitionStat
		usage     *disk.UsageStat
		want      bool
	}{
		{
			name:      "root filesystem",
			partition: disk.PartitionStat{Device: "/dev/sda1", Mountpoint: "/", Fstype: "ext4"},
			usage:     &disk.UsageStat{Total: 1024},
			want:      true,
		},
		{
			name:      "zero size pseudo filesystem",
			partition: disk.PartitionStat{Device: "devpts", Mountpoint: "/dev/pts", Fstype: "devpts"},
			usage:     &disk.UsageStat{Total: 0},
			want:      false,
		},
		{
			name:      "bind mount duplicate",
			partition: disk.PartitionStat{Device: "/dev/sda1", Mountpoint: "/data/bind", Fstype: "ext4", Opts: []string{"rw", "bind"}},
			usage:     &disk.UsageStat{Total: 1024},
			want:      false,
		},
		{
			name:      "docker overlay mount",
			partition: disk.PartitionStat{Device: "overlay", Mountpoint: "/var/lib/docker/overlay2/b11853e7551c9fcda21cac03f9197685865daa6d30ef9b106265b79b0637db2e/merged", Fstype: "overlay"},
			usage:     &disk.UsageStat{Total: 1024},
			want:      false,
		},
		{
			name:      "container root overlay",
			partition: disk.PartitionStat{Device: "overlay", Mountpoint: "/", Fstype: "overlay"},
			usage:     &disk.UsageStat{Total: 1024},
			want:      true,
		},
		{
			name:      "missing mountpoint",
			partition: disk.PartitionStat{Device: "/dev/sda1", Mountpoint: "", Fstype: "ext4"},
			usage:     &disk.UsageStat{Total: 1024},
			want:      false,
		},
		{
			name:      "device none",
			partition: disk.PartitionStat{Device: "none", Mountpoint: "/mnt/data", Fstype: "ext4"},
			usage:     &disk.UsageStat{Total: 1024},
			want:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isDisplayableDiskPartition(test.partition, test.usage)
			if got != test.want {
				t.Fatalf("isDisplayableDiskPartition() = %v, want %v", got, test.want)
			}
		})
	}
}
