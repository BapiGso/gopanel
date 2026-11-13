package monitor

import (
	"context"
	"errors"
	"math"
	"runtime"
	"slices"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"
)

type hostCounters struct {
	cpuTotal     float64
	cpuIdle      float64
	cpuKnown     bool
	memoryTotal  uint64
	memoryUsed   uint64
	memoryKnown  bool
	upload       uint64
	download     uint64
	networkKnown bool
}

type HostCollector struct {
	interfaces []string
	platform   string
	previous   hostCounters
	previousAt time.Time
}

func NewCollector(interfaces []string) *HostCollector {
	platform, _, _, _ := host.PlatformInformationWithContext(context.Background())
	return &HostCollector{interfaces: append([]string{}, interfaces...), platform: platform}
}

// Close is a no-op. The collector holds no background resources, but Service
// still closes it through the Collector interface.
func (collector *HostCollector) Close(context.Context) error {
	return nil
}

func (collector *HostCollector) Collect(ctx context.Context) (Metrics, error) {
	var counters hostCounters
	var failures []error
	if err := readCPU(ctx, &counters); err != nil {
		failures = append(failures, err)
	}
	if err := readMemory(ctx, &counters); err != nil {
		failures = append(failures, err)
	}
	if err := readNetwork(ctx, collector.interfaces, &counters); err != nil {
		failures = append(failures, err)
	}
	now := time.Now()
	metrics := countersToMetrics(counters, collector.previous, now.Sub(collector.previousAt), !collector.previousAt.IsZero())
	collector.previous, collector.previousAt = counters, now
	metrics.Platform, metrics.Architecture = collector.platform, runtime.GOARCH
	metrics.CPUCores = runtime.NumCPU()
	uptime, err := host.UptimeWithContext(ctx)
	if err != nil {
		failures = append(failures, err)
		metrics.UnavailableFields = append(metrics.UnavailableFields, "uptime")
	} else {
		metrics.UptimeSeconds = uptime
	}
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		failures = append(failures, err)
	}
	devices := map[string]bool{}
	for _, partition := range partitions {
		usage, usageError := disk.UsageWithContext(ctx, partition.Mountpoint)
		if usageError != nil || !isDisplayableDiskPartition(partition, usage) || devices[partition.Device] {
			continue
		}
		devices[partition.Device] = true
		metrics.DiskTotal += usage.Total
		metrics.DiskUsed += usage.Used
	}
	if metrics.DiskTotal == 0 {
		metrics.UnavailableFields = append(metrics.UnavailableFields, "disk")
	} else {
		metrics.DiskPercent = boundedPercent(float64(metrics.DiskUsed) / float64(metrics.DiskTotal) * 100)
	}
	metrics.Available = counters.cpuKnown || counters.memoryKnown || counters.networkKnown || metrics.DiskTotal != 0
	return metrics, errors.Join(failures...)
}

func readCPU(ctx context.Context, counters *hostCounters) error {
	times, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		return err
	}
	if len(times) != 1 {
		return errors.New("host CPU usage: incorrect summary count")
	}
	// Nice, iowait, interrupt, softirq, steal and guest time are grouped as
	// "other", matching the earlier OpenTelemetry host figures so CPU percentages stay comparable.
	hostTime := times[0]
	other := hostTime.Nice + hostTime.Iowait + hostTime.Irq + hostTime.Softirq + hostTime.Steal + hostTime.Guest + hostTime.GuestNice
	counters.cpuTotal = hostTime.User + hostTime.System + other + hostTime.Idle
	counters.cpuIdle = hostTime.Idle
	counters.cpuKnown = true
	return nil
}

func readMemory(ctx context.Context, counters *hostCounters) error {
	stats, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return err
	}
	counters.memoryTotal, counters.memoryUsed = stats.Total, stats.Used
	counters.memoryKnown = true
	return nil
}

func readNetwork(ctx context.Context, interfaces []string, counters *hostCounters) error {
	if len(interfaces) == 0 {
		stats, err := gopsnet.IOCountersWithContext(ctx, false)
		if err != nil {
			return err
		}
		if len(stats) != 1 {
			return errors.New("host network usage: incorrect summary count")
		}
		counters.upload, counters.download = stats[0].BytesSent, stats[0].BytesRecv
		counters.networkKnown = true
		return nil
	}
	stats, err := gopsnet.IOCountersWithContext(ctx, true)
	if err != nil {
		return err
	}
	for _, network := range stats {
		if slices.Contains(interfaces, network.Name) {
			counters.upload += network.BytesSent
			counters.download += network.BytesRecv
			counters.networkKnown = true
		}
	}
	return nil
}

func countersToMetrics(current, previous hostCounters, elapsed time.Duration, hasPrevious bool) Metrics {
	metrics := Metrics{UploadBytes: current.upload, DownloadBytes: current.download, MemoryUsed: current.memoryUsed, UnavailableFields: []string{}}
	if current.memoryKnown {
		metrics.MemoryTotal = current.memoryTotal
		if current.memoryTotal > 0 {
			metrics.MemoryPercent = boundedPercent(float64(current.memoryUsed) / float64(current.memoryTotal) * 100)
		}
	} else {
		metrics.UnavailableFields = append(metrics.UnavailableFields, "memory")
	}
	totalDelta := current.cpuTotal - previous.cpuTotal
	idleDelta := current.cpuIdle - previous.cpuIdle
	if hasPrevious && current.cpuKnown && previous.cpuKnown && totalDelta > 0 && idleDelta >= 0 && idleDelta <= totalDelta {
		metrics.CPUPercent = boundedPercent((totalDelta - idleDelta) / totalDelta * 100)
	} else {
		metrics.UnavailableFields = append(metrics.UnavailableFields, "cpu")
	}
	if hasPrevious && current.networkKnown && previous.networkKnown && elapsed > 0 && current.upload >= previous.upload && current.download >= previous.download {
		metrics.UploadRate = float64(current.upload-previous.upload) / elapsed.Seconds()
		metrics.DownloadRate = float64(current.download-previous.download) / elapsed.Seconds()
		metrics.RateAvailable = true
	} else {
		metrics.UnavailableFields = append(metrics.UnavailableFields, "network_rate")
	}
	if !current.networkKnown {
		metrics.UnavailableFields = append(metrics.UnavailableFields, "network")
	}
	return metrics
}

func boundedPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Max(0, math.Min(100, value))
}
