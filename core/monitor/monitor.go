package monitor

import (
	"path/filepath"
	"slices"
	"strings"
	"time" // sync removed

	"github.com/labstack/gommon/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// DiskUsageInfo 结构体保持不变
type DiskUsageInfo struct {
	Path        string  `json:"path"`
	Device      string  `json:"device"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

// Monitor 结构体移除 mu sync.RWMutex
type Monitor struct {
	CPUUsage  []float64              `json:"CPUUsage"`
	CPU       []cpu.InfoStat         `json:"CPU"`
	Memory    *mem.VirtualMemoryStat `json:"Memory"`
	Network   []net.IOCountersStat   `json:"Network"`
	DiskUsage []DiskUsageInfo        `json:"DiskUsage"`
	HostInfo  *host.InfoStat         `json:"HostInfo"`
	// mu        sync.RWMutex // 锁已移除
}

// 全局监视器实例
var SysMonitor = func() *Monitor {
	m := &Monitor{
		DiskUsage: []DiskUsageInfo{},
		CPU:       []cpu.InfoStat{},
		Network:   []net.IOCountersStat{},
		CPUUsage:  []float64{},
	}
	go m.fetchAllStats() // 启动后台刷新循环
	return m
}()

// fetchAllStats 收集所有系统统计信息，直接修改 Monitor 实例的字段
func (m *Monitor) fetchAllStats() {
	// 注意：这里不再有锁 m.mu.Lock() / m.mu.Unlock()
	var err error

	m.CPU, err = cpu.Info()
	if err != nil {
		log.Errorf("Error fetching CPU info: %v", err)
	}

	m.CPUUsage, err = cpu.Percent(time.Second, false)
	if err != nil {
		log.Errorf("Error fetching CPU percentage: %v", err)
	}

	m.Memory, err = mem.VirtualMemory()
	if err != nil {
		log.Errorf("Error fetching memory stats: %v", err)
	}

	m.Network, err = net.IOCounters(true)
	if err != nil {
		log.Errorf("Error fetching network stats: %v", err)
	}

	m.HostInfo, err = host.Info()
	if err != nil {
		log.Errorf("Error fetching host info: %v", err)
	}

	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Errorf("Failed to get disk partitions: %v", err)
		m.DiskUsage = []DiskUsageInfo{}
	} else {
		var diskInfos []DiskUsageInfo
		for _, partition := range partitions {
			usageStat, usageErr := disk.Usage(partition.Mountpoint)
			if usageErr != nil {
				continue
			}
			if !isDisplayableDiskPartition(partition, usageStat) {
				continue
			}
			diskInfos = append(diskInfos, DiskUsageInfo{
				Path:        usageStat.Path,
				Device:      partition.Device,
				Fstype:      usageStat.Fstype,
				Total:       usageStat.Total,
				Free:        usageStat.Free,
				Used:        usageStat.Used,
				UsedPercent: usageStat.UsedPercent,
			})
		}
		m.DiskUsage = diskInfos
	}
}

func isDisplayableDiskPartition(partition disk.PartitionStat, usage *disk.UsageStat) bool {
	if partition.Mountpoint == "" || partition.Device == "" || partition.Device == "none" {
		return false
	}
	if usage == nil || usage.Total == 0 {
		return false
	}
	if hasMountOption(partition, "bind") {
		return false
	}

	mountpoint := filepath.ToSlash(partition.Mountpoint)
	if strings.EqualFold(partition.Fstype, "overlay") && mountpoint != "/" {
		return false
	}
	return true
}

func hasMountOption(partition disk.PartitionStat, option string) bool {
	return slices.Contains(partition.Opts, option)
}
