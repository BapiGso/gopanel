package monitor

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

type record struct {
	report   Report
	lastSeen time.Time
	history  []Sample
	retired  []string
}

type Registry struct {
	mutex     sync.RWMutex
	nodes     map[string]*record
	offline   time.Duration
	history   int
	maximum   int
	validator *validator.Validate
}

func NewRegistry(offline time.Duration, history, maximum int) *Registry {
	return &Registry{
		nodes: map[string]*record{}, offline: offline, history: history, maximum: maximum,
		validator: validator.New(),
	}
}

func (registry *Registry) Accept(report Report, received time.Time) error {
	if err := registry.validator.Struct(report); err != nil {
		return fmt.Errorf("invalid metric report: %w", err)
	}
	if strings.ContainsAny(report.Node.ID, "/\\?#% \t\r\n") {
		return fmt.Errorf("node.id must be URL-safe")
	}
	if (report.Node.Latitude == nil) != (report.Node.Longitude == nil) {
		return fmt.Errorf("node coordinates must be supplied together")
	}
	if report.Metrics.MemoryUsed > report.Metrics.MemoryTotal || report.Metrics.DiskUsed > report.Metrics.DiskTotal {
		return fmt.Errorf("used memory and disk space must not exceed their totals")
	}
	if report.CollectedAt.IsZero() || report.CollectedAt.After(received.Add(2*time.Minute)) || report.CollectedAt.Before(received.Add(-10*time.Minute)) {
		return fmt.Errorf("collected_at is outside the accepted clock-skew window")
	}
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	current := registry.nodes[report.Node.ID]
	if current == nil {
		if len(registry.nodes) >= registry.maximum {
			return fmt.Errorf("monitor node limit reached")
		}
		current = &record{history: []Sample{}, retired: []string{}}
		registry.nodes[report.Node.ID] = current
	}
	if current.report.BootID == report.BootID && report.Sequence <= current.report.Sequence {
		return nil
	}
	for _, retired := range current.retired {
		if retired == report.BootID {
			return nil
		}
	}
	if current.report.BootID != "" && current.report.BootID != report.BootID {
		if !report.CollectedAt.After(current.report.CollectedAt) {
			return nil
		}
		current.retired = append(current.retired, current.report.BootID)
		if len(current.retired) > 8 {
			current.retired = current.retired[len(current.retired)-8:]
		}
	}
	report.Node = cloneNode(report.Node)
	report.Metrics.UnavailableFields = append([]string{}, report.Metrics.UnavailableFields...)
	current.report = report
	current.lastSeen = received
	current.history = append(current.history, Sample{
		Time: report.CollectedAt, CPUPercent: report.Metrics.CPUPercent, MemoryPercent: report.Metrics.MemoryPercent,
		UploadRate: report.Metrics.UploadRate, DownloadRate: report.Metrics.DownloadRate,
	})
	if len(current.history) > registry.history {
		copy(current.history, current.history[len(current.history)-registry.history:])
		current.history = current.history[:registry.history]
	}
	return nil
}

func (registry *Registry) Views(now time.Time) []NodeView {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	views := make([]NodeView, 0, len(registry.nodes))
	for _, current := range registry.nodes {
		views = append(views, registry.view(current, now))
	}
	sort.Slice(views, func(first, second int) bool { return views[first].Name < views[second].Name })
	return views
}

func (registry *Registry) Node(id string, now time.Time) (NodeView, bool) {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	current, exists := registry.nodes[id]
	if !exists {
		return NodeView{}, false
	}
	return registry.view(current, now), true
}

func (registry *Registry) History(id string) ([]Sample, bool) {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	current, exists := registry.nodes[id]
	if !exists {
		return nil, false
	}
	return append([]Sample{}, current.history...), true
}

func (registry *Registry) view(current *record, now time.Time) NodeView {
	metrics := current.report.Metrics
	metrics.UnavailableFields = append([]string{}, metrics.UnavailableFields...)
	return NodeView{
		Node: cloneNode(current.report.Node), Metrics: metrics, Online: now.Sub(current.lastSeen) <= registry.offline,
		CollectedAt: current.report.CollectedAt, LastSeen: current.lastSeen,
	}
}
