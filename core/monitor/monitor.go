package monitor

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"time"
)

func init() {
	M.refresh()
}

type Monitor struct {
	Cpu      []float64
	Memy     *mem.VirtualMemoryStat
	Diskpart []disk.PartitionStat
	Ysinfo   *host.InfoStat
	Livetime string
}

var (
	M = &Monitor{}
)

func (m *Monitor) refresh() {
	var err error
	m.Memy, err = mem.VirtualMemory()
	if err != nil {
		log.Fatal(err)
	}
	//
	m.Cpu, err = cpu.Percent(time.Duration(3)*time.Second, false)
	if err != nil {
		log.Fatal(err)
	}
	//
	m.Diskpart, err = disk.Partitions(false)
	if err != nil {
		log.Fatal(err)
	}
	m.Ysinfo, err = host.Info()
	if err != nil {
		log.Fatal(err)
	}
}

func (m *Monitor) GetUptime() string {
	t := time.Unix(int64(m.Ysinfo.Uptime), 0)
	// 计算当前时间与指定时间的时间差
	duration := time.Since(t)
	// 输出时间差，按天、小时、分钟、秒的顺序依次计算
	days := int(duration.Hours() / 24 / 60 / 60)
	hours := int(duration.Hours()) % 24
	return fmt.Sprintf("%d天%d小时", days, hours)
}
