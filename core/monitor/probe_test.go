package monitor

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gopanel/core/storage"
)

func TestCompactMonitoringConfigInfersRolesAndPersistsOnlyIdentity(t *testing.T) {
	key := strings.Repeat("k", 32)
	for _, example := range []struct {
		name    string
		content string
		role    string
	}{
		{"probe", `{"key":"` + key + `"}`, "probe"},
		{"master", `{"key":"` + key + `","peers":["192.0.2.10","2001:db8::1"]}`, "master"},
		{"existing panel", `{"role":"standalone","key":"` + key + `"}`, "standalone"},
	} {
		t.Run(example.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, ConfigFileName)
			if err := storage.AtomicWrite(path, []byte(example.content)); err != nil {
				t.Fatal(err)
			}
			first, err := LoadConfig(directory)
			if err != nil {
				t.Fatal(err)
			}
			second, err := LoadConfig(directory)
			if err != nil || first.NodeID != second.NodeID || first.Role != example.role || first.Name != "Node-"+first.NodeID[:8] {
				t.Fatalf("compact config lost its defaults or identity: %+v %v", first, err)
			}
			document, err := (&storage.File{Path: path}).Read()
			if err != nil {
				t.Fatal(err)
			}
			var original, saved map[string]any
			if err := json.Unmarshal([]byte(example.content), &original); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(document.Content), &saved); err != nil {
				t.Fatal(err)
			}
			if len(saved) != len(original)+1 || saved["node_id"] != first.NodeID {
				t.Fatal("loading compact config wrote unnecessary defaults")
			}
		})
	}
}

func TestConcurrentCompactConfigLoadsKeepOneIdentity(t *testing.T) {
	directory := t.TempDir()
	if err := storage.AtomicWrite(filepath.Join(directory, ConfigFileName), []byte(`{"key":"`+strings.Repeat("k", 32)+`"}`)); err != nil {
		t.Fatal(err)
	}
	identities := make(chan string, 12)
	failures := make(chan error, 12)
	var readers sync.WaitGroup
	for range 12 {
		readers.Go(func() {
			configuration, err := LoadConfig(directory)
			if err != nil {
				failures <- err
				return
			}
			identities <- configuration.NodeID
		})
	}
	readers.Wait()
	close(failures)
	close(identities)
	for err := range failures {
		t.Fatal(err)
	}
	identity := ""
	for current := range identities {
		if identity != "" && identity != current {
			t.Fatal("concurrent loads persisted different node identities")
		}
		identity = current
	}
}

func TestCompactConfigRejectsInvalidOrMixedInputs(t *testing.T) {
	key := strings.Repeat("k", 32)
	for _, content := range []string{
		`{"key":"short"}`,
		`{"key":"` + key + `","node_id":""}`,
		`{"key":"` + key + `","node_id":null}`,
		`{"key":"` + key + `","unknown":true}`,
		`{"key":"` + key + `","peers":["192.0.2.10","192.0.2.10:9555"]}`,
		`{"key":"` + key + `","peers":["example.com"]}`,
		`{"key":"` + key + `","peers":["https://example.com/path"]}`,
		`{"key":"` + key + `","peers":["192.0.2.10:0"]}`,
		`{"key":"` + key + `","peers":["192.0.2.10"],"role":"probe"}`,
		`{"key":"` + key + `","peers":["192.0.2.10"],"cluster_id":"legacy"}`,
		`{"key":"` + key + `","role":"master"}`,
		`{"key":"` + key + `","listen":"9555"}`,
	} {
		if _, err := parseConfig([]byte(content)); err == nil {
			t.Fatalf("accepted invalid compact configuration: %s", content)
		}
	}
	for input, expected := range map[string]string{
		"192.0.2.10":            "http://192.0.2.10:9555",
		"192.0.2.10:9666":       "http://192.0.2.10:9666",
		"2001:db8::1":           "http://[2001:db8::1]:9555",
		"[2001:db8::1]:9666":    "http://[2001:db8::1]:9666",
		"https://example.com":   "https://example.com",
		"http://127.0.0.1:1234": "http://127.0.0.1:1234",
	} {
		address, err := peerAddress(input)
		if err != nil || address != expected {
			t.Fatalf("peerAddress(%s) = %s, %v; want %s", input, address, err, expected)
		}
	}
}

func testPullProbe(t *testing.T, identity string) (*Service, *httptest.Server) {
	t.Helper()
	configuration := DefaultConfig()
	configuration.Role, configuration.Key, configuration.NodeID, configuration.Name = "probe", strings.Repeat("k", 32), identity, identity
	service, err := NewService(configuration, &fakeCollector{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.sample(context.Background())
	server := httptest.NewServer(service.probeRoutes())
	t.Cleanup(func() { server.Close(); service.Close(context.Background()) })
	return service, server
}

func testPullMaster(t *testing.T, addresses ...string) *Service {
	t.Helper()
	configuration := DefaultConfig()
	configuration.Role, configuration.Key, configuration.Name = "master", strings.Repeat("k", 32), "Manual master"
	configuration.Peers = addresses
	service, err := NewService(configuration, &fakeCollector{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.sample(context.Background())
	t.Cleanup(func() { service.Close(context.Background()) })
	return service
}

func TestManualMasterPullsEveryPeerWithoutElection(t *testing.T) {
	first, firstServer := testPullProbe(t, "first-probe-123")
	_, secondServer := testPullProbe(t, "second-probe-123")
	master := testPullMaster(t, firstServer.URL, secondServer.URL)
	master.pullPeers(context.Background())
	view := master.Overview()
	if view.Total != 3 || view.Online != 3 || view.Cluster.Coordination != "manual" || view.Cluster.LeaderID != master.settings.NodeID || view.Cluster.HealthyPeers != 2 || view.Cluster.PeerCount != 2 {
		t.Fatalf("manual aggregation failed: %+v", view)
	}
	before, _ := first.registry.History(first.settings.NodeID)
	master.pullPeers(context.Background())
	after, _ := first.registry.History(first.settings.NodeID)
	if len(before) != len(after) {
		t.Fatal("a master's request resampled its probe")
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{master.settings.Key, firstServer.URL, secondServer.URL} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("aggregate view exposed a key or peer address")
		}
	}
	firstServer.Close()
	master.mutex.Lock()
	peer := master.peers[firstServer.URL]
	peer.LastSeen = time.Now().Add(-time.Minute)
	master.peers[firstServer.URL] = peer
	master.mutex.Unlock()
	if view := master.Cluster(time.Now()); view.LeaderID != master.settings.NodeID || view.HealthyPeers != 1 || !view.Degraded {
		t.Fatal("a failed probe changed the manually selected master")
	}
}

func TestProbeReadEndpointRequiresFreshIdentityAndHasNoAdministration(t *testing.T) {
	_, server := testPullProbe(t, "readonly-probe-123")
	for _, example := range []struct {
		method    string
		path      string
		key       string
		timestamp time.Time
		want      int
	}{
		{http.MethodGet, probeReportPath, "", time.Now(), http.StatusUnauthorized},
		{http.MethodGet, probeReportPath, "wrong", time.Now(), http.StatusUnauthorized},
		{http.MethodGet, probeReportPath, strings.Repeat("k", 32), time.Now().Add(-2 * time.Minute), http.StatusUnauthorized},
		{http.MethodGet, probeReportPath, strings.Repeat("k", 32), time.Now(), http.StatusOK},
		{http.MethodGet, probeReportPath + "?command=exec", strings.Repeat("k", 32), time.Now(), http.StatusBadRequest},
		{http.MethodPost, probeReportPath, strings.Repeat("k", 32), time.Now(), http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/v1/caddy/stop", strings.Repeat("k", 32), time.Now(), http.StatusNotFound},
		{http.MethodGet, "/admin/security", strings.Repeat("k", 32), time.Now(), http.StatusNotFound},
	} {
		request, err := http.NewRequest(example.method, server.URL+example.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		timestamp, nonce := strconv.FormatInt(example.timestamp.Unix(), 10), "test-request-nonce-123"
		request.Header.Set(probeTimeHeader, timestamp)
		request.Header.Set(probeNonceHeader, nonce)
		if example.key != "" {
			request.Header.Set(probeSignatureHeader, probeRequestSignature(example.key, timestamp, nonce))
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != example.want {
			t.Fatalf("%s %s = %d, want %d: %s", example.method, example.path, response.StatusCode, example.want, body)
		}
		if response.StatusCode == http.StatusOK && !sameSecret(response.Header.Get(probeSignatureHeader), probeResponseSignature(example.key, timestamp, nonce, body)) {
			t.Fatal("probe returned an unsigned snapshot")
		}
		if strings.Contains(string(body), strings.Repeat("k", 32)) {
			t.Fatal("probe response exposed its key")
		}
	}
}

func TestMasterRejectsTamperingReplayAndOversizedResponses(t *testing.T) {
	key := strings.Repeat("k", 32)
	for _, mode := range []string{"wrong key", "tampered", "old nonce", "oversized", "command"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				report := validReport(time.Now())
				content, _ := json.Marshal(map[string]any{"data": report})
				if mode == "oversized" {
					content = append(content, []byte(strings.Repeat(" ", 32<<10))...)
				}
				if mode == "command" {
					content = []byte(`{"data":{},"command":"stop"}`)
				}
				signingKey, nonce := key, request.Header.Get(probeNonceHeader)
				if mode == "wrong key" {
					signingKey = strings.Repeat("x", 32)
				}
				if mode == "old nonce" {
					nonce = "old-request-nonce-123"
				}
				writer.Header().Set(probeSignatureHeader, probeResponseSignature(signingKey, request.Header.Get(probeTimeHeader), nonce, content))
				if mode == "tampered" {
					content = append(content, ' ')
				}
				writer.Write(content)
			}))
			defer server.Close()
			master := testPullMaster(t, server.URL)
			if _, err := master.pull(context.Background(), server.URL); err == nil {
				t.Fatal("master trusted an invalid probe response")
			}
			if master.Overview().Total != 1 {
				t.Fatal("invalid response changed the registry")
			}
		})
	}
}

func TestMasterRejectsDuplicateAndChangedNodeIdentities(t *testing.T) {
	first, firstServer := testPullProbe(t, "same-probe-123")
	_, secondServer := testPullProbe(t, "same-probe-123")
	master := testPullMaster(t, firstServer.URL, secondServer.URL)
	report, err := master.pull(context.Background(), firstServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := master.acceptPeer(firstServer.URL, report); err != nil {
		t.Fatal(err)
	}
	if err := master.acceptPeer(secondServer.URL, report); err == nil {
		t.Fatal("two peer IPs shared one node identity")
	}
	report.Node.ID = "replacement-probe-123"
	if err := master.acceptPeer(firstServer.URL, report); err == nil {
		t.Fatal("a replaced node identity was silently accepted")
	}
	if err := master.acceptPeer(firstServer.URL, first.latest); err != nil {
		t.Fatal(err)
	}
}

func TestPullCancellationAndProbeListenerShutdown(t *testing.T) {
	entered := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(entered)
		<-request.Context().Done()
	}))
	defer server.Close()
	master := testPullMaster(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { master.pullPeers(ctx); close(done) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("master did not start reading its peer")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelled master retained an active peer request")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	configuration := DefaultConfig()
	configuration.Role, configuration.Key, configuration.Listen = "probe", strings.Repeat("k", 32), address
	collector := &fakeCollector{}
	probe, err := NewService(configuration, collector, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { probe.Close(context.Background()) })
	if err := probe.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for collector.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	shutdown, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := probe.Close(shutdown); err != nil || !collector.closed.Load() {
		t.Fatalf("probe shutdown failed: %v", err)
	}
	connection, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("stopped probe retained its HTTP listener")
	}
}
