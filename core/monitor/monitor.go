package monitor

import (
	"github.com/labstack/gommon/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"time"
)

type Monitor struct {
	CPUUsage []float64
	CPU      []cpu.InfoStat
	Memory   *mem.VirtualMemoryStat
	Network  []net.IOCountersStat
	Diskpart map[string]disk.IOCountersStat
	HostInfo *host.InfoStat
}

var M = &Monitor{}

func (m *Monitor) refresh() {
	var err error
	go func() {
		m.Memory, err = mem.VirtualMemory()
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		m.Network, err = net.IOCounters(true)
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		m.CPU, err = cpu.Info()
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		m.CPUUsage, err = cpu.Percent(time.Duration(3)*time.Second, false)
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		m.Diskpart, err = disk.IOCounters("")
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		m.HostInfo, err = host.Info()
		if err != nil {
			log.Fatal(err)
		}
	}()
}
