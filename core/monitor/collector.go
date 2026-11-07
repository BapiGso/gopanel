package monitor

import (
	"context"
	"errors"
	"math"
	"runtime"
	"slices"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	gopsnet "github.com/shirou/gopsutil/v4/net"
	otelhost "go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type hostCounters struct {
	cpuTotal        float64
	cpuIdle         float64
	cpuKnown        bool
	memoryUsed      uint64
	memoryFree      uint64
	memoryRatio     float64
	memoryFreeRatio float64
	memoryKnown     bool
	upload          uint64
	download        uint64
	networkKnown    bool
}

type HostCollector struct {
	reader     *sdkmetric.ManualReader
	provider   *sdkmetric.MeterProvider
	interfaces []string
	platform   string
	previous   hostCounters
	previousAt time.Time
}

func NewCollector(interfaces []string) (*HostCollector, error) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	if err := otelhost.Start(otelhost.WithMeterProvider(provider)); err != nil {
		provider.Shutdown(context.Background())
		return nil, err
	}
	platform, _, _, _ := host.PlatformInformationWithContext(context.Background())
	return &HostCollector{reader: reader, provider: provider, interfaces: append([]string{}, interfaces...), platform: platform}, nil
}

func (collector *HostCollector) Close(ctx context.Context) error {
	return collector.provider.Shutdown(ctx)
}

func (collector *HostCollector) Collect(ctx context.Context) (Metrics, error) {
	var data metricdata.ResourceMetrics
	var failures []error
	if err := collector.reader.Collect(ctx, &data); err != nil {
		failures = append(failures, err)
	}
	counters := decodeHostCounters(data)
	if len(collector.interfaces) != 0 {
		interfaces, err := gopsnet.IOCountersWithContext(ctx, true)
		counters.networkKnown = false
		counters.upload, counters.download = 0, 0
		if err != nil {
			failures = append(failures, err)
		} else {
			for _, network := range interfaces {
				if slices.Contains(collector.interfaces, network.Name) {
					counters.upload += network.BytesSent
					counters.download += network.BytesRecv
					counters.networkKnown = true
				}
			}
		}
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

func decodeHostCounters(data metricdata.ResourceMetrics) hostCounters {
	var counters hostCounters
	for _, scope := range data.ScopeMetrics {
		if scope.Scope.Name != otelhost.ScopeName {
			continue
		}
		for _, measurement := range scope.Metrics {
			switch measurement.Name {
			case "system.cpu.time":
				if total, ok := measurement.Data.(metricdata.Sum[float64]); ok {
					for _, point := range total.DataPoints {
						counters.cpuTotal += point.Value
						if attributeText(point.Attributes, "cpu.mode") == "idle" {
							counters.cpuIdle += point.Value
						}
					}
					counters.cpuKnown = len(total.DataPoints) != 0
				}
			case "system.memory.usage":
				if usage, ok := measurement.Data.(metricdata.Sum[int64]); ok {
					for _, point := range usage.DataPoints {
						if point.Value >= 0 {
							switch attributeText(point.Attributes, "system.memory.state") {
							case "used":
								counters.memoryUsed = uint64(point.Value)
							case "free":
								counters.memoryFree = uint64(point.Value)
							}
						}
					}
				}
			case "system.memory.utilization":
				if utilization, ok := measurement.Data.(metricdata.Gauge[float64]); ok {
					for _, point := range utilization.DataPoints {
						switch attributeText(point.Attributes, "system.memory.state") {
						case "used":
							counters.memoryRatio = point.Value
						case "free":
							counters.memoryFreeRatio = point.Value
						}
					}
					counters.memoryKnown = len(utilization.DataPoints) != 0
				}
			case "system.network.io":
				if network, ok := measurement.Data.(metricdata.Sum[int64]); ok {
					for _, point := range network.DataPoints {
						if point.Value >= 0 {
							switch attributeText(point.Attributes, "network.io.direction") {
							case "transmit":
								counters.upload += uint64(point.Value)
							case "receive":
								counters.download += uint64(point.Value)
							}
						}
					}
					counters.networkKnown = len(network.DataPoints) != 0
				}
			}
		}
	}
	return counters
}

func attributeText(attributes attribute.Set, key string) string {
	value, _ := attributes.Value(attribute.Key(key))
	return value.AsString()
}

func countersToMetrics(current, previous hostCounters, elapsed time.Duration, hasPrevious bool) Metrics {
	metrics := Metrics{UploadBytes: current.upload, DownloadBytes: current.download, MemoryUsed: current.memoryUsed, UnavailableFields: []string{}}
	if current.memoryKnown {
		metrics.MemoryPercent = boundedPercent(current.memoryRatio * 100)
		if current.memoryRatio > 0 {
			metrics.MemoryTotal = uint64(math.Round(float64(current.memoryUsed) / current.memoryRatio))
		} else if current.memoryFreeRatio > 0 {
			metrics.MemoryTotal = uint64(math.Round(float64(current.memoryFree) / current.memoryFreeRatio))
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
