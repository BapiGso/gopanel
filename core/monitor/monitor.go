package monitor

import (
	"github.com/labstack/gommon/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
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

var M = func() *Monitor {
	tmp := &Monitor{}
	go tmp.refresh()
	return tmp
}()

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
