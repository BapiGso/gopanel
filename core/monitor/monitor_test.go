package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
	"gopanel/core/storage"
)

type fakeCollector struct {
	calls  atomic.Uint64
	closed atomic.Bool
}

func (collector *fakeCollector) Collect(ctx context.Context) (Metrics, error) {
	sequence := collector.calls.Add(1)
	return Metrics{
		CPUPercent: 12, MemoryPercent: 25, MemoryTotal: 1024, MemoryUsed: 256,
		DiskPercent: 20, DiskTotal: 2048, DiskUsed: 410,
		UploadBytes: sequence * 100, DownloadBytes: sequence * 200, UploadRate: 50, DownloadRate: 100,
		UptimeSeconds: 3600, Platform: "test-os", Architecture: "amd64", CPUCores: 2,
		Available: true, RateAvailable: true, UnavailableFields: []string{},
	}, nil
}

func (collector *fakeCollector) Close(ctx context.Context) error {
	collector.closed.Store(true)
	return nil
}

func validReport(now time.Time) Report {
	return Report{
		Node:   Node{ID: "test-node-123", Name: "Test node", Country: "DE", Tags: []string{"Service"}},
		BootID: "test-boot-123", Sequence: 1, CollectedAt: now,
		Metrics: Metrics{CPUPercent: 10, MemoryPercent: 25, MemoryTotal: 1024, MemoryUsed: 256, Available: true, UnavailableFields: []string{}},
	}
}

func TestRegistryDeduplicatesAndExpiresByReceiptTime(t *testing.T) {
	now := time.Now()
	registry := NewRegistry(5*time.Second, 3, 10)
	report := validReport(now)
	if err := registry.Accept(report, now); err != nil {
		t.Fatal(err)
	}
	if err := registry.Accept(report, now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	view, exists := registry.Node(report.Node.ID, now.Add(6*time.Second))
	if !exists || view.Online || !view.LastSeen.Equal(now) {
		t.Fatal("replaying the same sequence extended the online lifetime")
	}
	history, _ := registry.History(report.Node.ID)
	if len(history) != 1 {
		t.Fatal("duplicate report appended history")
	}
	for sequence := uint64(2); sequence <= 5; sequence++ {
		report.Sequence = sequence
		report.CollectedAt = now.Add(time.Duration(sequence) * time.Second)
		if err := registry.Accept(report, report.CollectedAt); err != nil {
			t.Fatal(err)
		}
	}
	history, _ = registry.History(report.Node.ID)
	if len(history) != 3 || !history[0].Time.Equal(now.Add(3*time.Second)) {
		t.Fatal("history is not a bounded newest-samples window")
	}
}

func TestRegistryRejectsOldBootsAndReturnsIndependentViews(t *testing.T) {
	now := time.Now()
	registry := NewRegistry(time.Minute, 3, 1)
	report := validReport(now)
	latitude, longitude := 50.0, 8.0
	report.Node.Latitude, report.Node.Longitude = &latitude, &longitude
	if err := registry.Accept(report, now); err != nil {
		t.Fatal(err)
	}
	report.Node.Tags[0] = "mutated"
	latitude = 10
	view, _ := registry.Node(report.Node.ID, now)
	if view.Tags[0] != "Service" || *view.Latitude != 50 {
		t.Fatal("registry retains mutable input references")
	}
	view.Tags[0] = "changed view"
	*view.Latitude = 0
	view, _ = registry.Node(report.Node.ID, now)
	if view.Tags[0] != "Service" || *view.Latitude != 50 {
		t.Fatal("mutating a returned view changed registry state")
	}
	restarted := validReport(now.Add(time.Second))
	restarted.BootID = "new-boot-123"
	if err := registry.Accept(restarted, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	report.Sequence = 999
	report.CollectedAt = now.Add(2 * time.Second)
	if err := registry.Accept(report, report.CollectedAt); err != nil {
		t.Fatal(err)
	}
	view, _ = registry.Node(report.Node.ID, now.Add(2*time.Second))
	if !view.CollectedAt.Equal(restarted.CollectedAt) {
		t.Fatal("retired boot replay replaced current metrics")
	}
	extra := validReport(now)
	extra.Node.ID = "another-node-123"
	if err := registry.Accept(extra, now); err == nil {
		t.Fatal("registry exceeded its node capacity")
	}
}

func TestCountersUseElapsedTimeAndHandleResets(t *testing.T) {
	previous := hostCounters{cpuTotal: 100, cpuIdle: 80, cpuKnown: true, memoryKnown: true, memoryUsed: 256, memoryRatio: .25, upload: 1000, download: 2000, networkKnown: true}
	current := previous
	current.cpuTotal, current.cpuIdle = 110, 85
	current.upload, current.download = 1400, 3000
	metrics := countersToMetrics(current, previous, 2*time.Second, true)
	if metrics.CPUPercent != 50 || metrics.UploadRate != 200 || metrics.DownloadRate != 500 || metrics.MemoryTotal != 1024 || !metrics.RateAvailable {
		t.Fatalf("incorrect counter conversion: %+v", metrics)
	}
	current.upload = 1
	current.cpuTotal = 1
	metrics = countersToMetrics(current, previous, time.Second, true)
	if metrics.RateAvailable || metrics.UploadRate != 0 || metrics.CPUPercent != 0 {
		t.Fatal("counter reset caused an unsigned-underflow spike")
	}
	metrics = countersToMetrics(previous, hostCounters{}, time.Second, false)
	if metrics.RateAvailable || metrics.CPUPercent != 0 {
		t.Fatal("first sample invented a rate without a baseline")
	}
}

func TestHostCollectorUsesOpenTelemetryAndSanitizedMetrics(t *testing.T) {
	collector, err := NewCollector(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer collector.Close(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	metrics, err := collector.Collect(ctx)
	if err != nil {
		t.Logf("partial platform metrics: %v", err)
	}
	if metrics.MemoryTotal == 0 || metrics.MemoryUsed > metrics.MemoryTotal || metrics.CPUCores < 1 || metrics.Architecture == "" {
		t.Fatalf("OpenTelemetry host data was not decoded: %+v", metrics)
	}
	encoded, err := json.Marshal(metrics)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateField := range []string{"hostname", "ip_address", "processes", "mountpoint", "device"} {
		if strings.Contains(string(encoded), `"`+privateField+`"`) {
			t.Fatalf("public metrics contain %s", privateField)
		}
	}
}

func TestConfigIdentityIsStableAndEndpointsAreExplicit(t *testing.T) {
	directory := t.TempDir()
	first, err := LoadConfig(directory)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadConfig(directory)
	if err != nil || first.NodeID != second.NodeID {
		t.Fatal("loading configuration regenerated the node identity")
	}
	configuration := DefaultConfig()
	configuration.Role = "candidate"
	configuration.ClusterID = "test-cluster"
	configuration.ClusterKey = strings.Repeat("k", 32)
	configuration.Candidates = []string{"https://example.com"}
	if err := configuration.Validate(); err == nil {
		t.Fatal("candidate without an advertised member URL was accepted")
	}
	configuration.AdvertiseURL = "https://example.com"
	if err := configuration.Validate(); err != nil {
		t.Fatal(err)
	}
	configuration.Candidates[0] = "http://example.com"
	configuration.AdvertiseURL = "http://example.com"
	if err := configuration.Validate(); err == nil {
		t.Fatal("public HTTP transport was silently accepted")
	}
	if err := storage.AtomicWrite(filepath.Join(directory, ConfigFileName), []byte(`{"unknown":true}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(directory); err == nil {
		t.Fatal("invalid existing configuration was reset instead of rejected")
	}
}

func TestReadsDoNotResampleAndClosingStopsCollection(t *testing.T) {
	configuration := DefaultConfig()
	configuration.SampleSeconds, configuration.OfflineSeconds = 60, 240
	collector := &fakeCollector{}
	service, err := NewService(configuration, collector, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.Start(context.Background())
	deadline := time.Now().Add(time.Second)
	for collector.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	for range 1000 {
		service.Overview()
	}
	if collector.calls.Load() != 1 {
		t.Fatal("reading cached views triggered new collection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Close(ctx); err != nil || !collector.closed.Load() {
		t.Fatalf("collector shutdown failed: %v", err)
	}
}

func testCandidate(t *testing.T, id string, addresses []string) (*Service, *httptest.Server) {
	t.Helper()
	server := echo.New()
	server.HTTPErrorHandler = httpapi.ErrorHandler
	listener := httptest.NewUnstartedServer(server)
	configuration := DefaultConfig()
	configuration.Role, configuration.NodeID, configuration.Name = "candidate", id, id
	configuration.SampleSeconds = 1
	configuration.ClusterID, configuration.ClusterKey = "test-cluster", strings.Repeat("k", 32)
	configuration.AdvertiseURL = "http://" + listener.Listener.Addr().String()
	configuration.Candidates = append(addresses, configuration.AdvertiseURL)
	service, err := NewService(configuration, &fakeCollector{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.RegisterAPI(httpapi.NewRouter(server.Group(httpapi.Prefix)), server)
	listener.Start()
	t.Cleanup(func() { listener.Close(); service.Close(context.Background()) })
	return service, listener
}

func TestAuthenticatedProbeFanoutWarmsMultipleCandidates(t *testing.T) {
	first, firstServer := testCandidate(t, "candidate-one", nil)
	second, secondServer := testCandidate(t, "candidate-two", []string{firstServer.URL})
	configuration := DefaultConfig()
	configuration.Role, configuration.Name = "probe", "Probe"
	configuration.SampleSeconds = 1
	configuration.ClusterID, configuration.ClusterKey = "test-cluster", strings.Repeat("k", 32)
	configuration.Candidates = []string{firstServer.URL, secondServer.URL}
	probe, err := NewService(configuration, &fakeCollector{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	probe.Start(context.Background())
	t.Cleanup(func() { probe.Close(context.Background()) })
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		_, firstHas := first.registry.Node(configuration.NodeID, time.Now())
		_, secondHas := second.registry.Node(configuration.NodeID, time.Now())
		if firstHas && secondHas && probe.Cluster(time.Now()).HealthyCandidates == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if first.Overview().Total != 1 || second.Overview().Total != 1 || probe.Cluster(time.Now()).LeaderID != "candidate-one" {
		t.Fatal("probe did not populate every candidate or choose the ordered preferred aggregator")
	}
	firstServer.Close()
	probe.mutex.Lock()
	peer := probe.peers[firstServer.URL]
	peer.LastSeen = time.Now().Add(-time.Minute)
	probe.peers[firstServer.URL] = peer
	probe.mutex.Unlock()
	if view := probe.Cluster(time.Now()); view.LeaderID != "candidate-two" || !view.Degraded {
		t.Fatal("expired preferred aggregator did not fail over to the warmed candidate")
	}
}

func TestIngestRequiresSeparateCredentialsAndRejectsCommands(t *testing.T) {
	service, server := testCandidate(t, "candidate-one", nil)
	report := validReport(time.Now())
	content, _ := json.Marshal(report)
	for _, example := range []struct {
		body string
		key  string
		want int
	}{
		{string(content), "", http.StatusUnauthorized},
		{string(content), "wrong", http.StatusUnauthorized},
		{strings.TrimSuffix(string(content), "}") + `,"command":"shutdown"}`, strings.Repeat("k", 32), http.StatusBadRequest},
		{string(content), strings.Repeat("k", 32), http.StatusOK},
	} {
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/monitor/reports", strings.NewReader(example.body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+example.key)
		request.Header.Set("X-GoPanel-Cluster", "test-cluster")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != example.want {
			t.Fatalf("ingest = %d %s, want %d", response.StatusCode, body, example.want)
		}
	}
	if service.Overview().Total != 1 {
		t.Fatal("invalid requests changed the registry")
	}
	response, err := http.Get(server.URL + "/api/v1/public/monitor/nodes")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || strings.Contains(string(body), strings.Repeat("k", 32)) || strings.Contains(string(body), server.URL) {
		t.Fatal("public response is unavailable or exposes credentials / candidate URLs")
	}
}

func TestStreamCancellationReleasesViewerSlot(t *testing.T) {
	service, server := testCandidate(t, "candidate-one", nil)
	ctx, cancel := context.WithCancel(context.Background())
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/public/monitor/events", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	buffer := make([]byte, 16)
	if _, err := response.Body.Read(buffer); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	response.Body.Close()
	deadline := time.Now().Add(time.Second)
	for len(service.streams) != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(service.streams) != 0 {
		t.Fatal("disconnected SSE viewer retained a stream slot")
	}
}

func ExampleConfig() {
	configuration := DefaultConfig()
	fmt.Println(configuration.Role, configuration.Public, configuration.SampleSeconds)
}
