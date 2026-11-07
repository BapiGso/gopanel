package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopanel/core/config"
	"gopanel/core/monitor"
	"gopanel/core/storage"
)

func TestPublicMonitorDoesNotGrantPanelOrProbePermissions(t *testing.T) {
	useTempCoreConfig(t, config.Config{
		Panel: config.PanelConfig{Port: ":8443", Path: "login", Username: "paneluser", Password: "panelpass123"},
	})
	settings := monitor.DefaultConfig()
	settings.Role = "candidate"
	settings.ClusterID, settings.ClusterKey = "test-cluster", strings.Repeat("k", 32)
	settings.AdvertiseURL = "https://localhost:8443"
	settings.Candidates = []string{settings.AdvertiseURL}
	content, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.AtomicWrite(filepath.Join(config.Directory(), monitor.ConfigFileName), content); err != nil {
		t.Fatal(err)
	}
	panel := New()
	t.Cleanup(func() { panel.monitoring.Close(context.Background()) })
	panel.setupRoutes()
	report := monitor.Report{
		Node: monitor.Node{ID: "external-probe-123", Name: "Public alias", Country: "DE", Tags: []string{"Relay"}},
		BootID: "test-boot-123", Sequence: 1, CollectedAt: time.Now().UTC(),
		Metrics: monitor.Metrics{CPUPercent: 10, MemoryPercent: 25, MemoryTotal: 1024, MemoryUsed: 256, Available: true, UnavailableFields: []string{}},
	}
	content, _ = json.Marshal(report)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/monitor/reports", strings.NewReader(string(content)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+settings.ClusterKey)
	request.Header.Set("X-GoPanel-Cluster", settings.ClusterID)
	response := httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authorized probe report = %d %s", response.Code, response.Body.String())
	}
	for _, path := range []string{"/api/v1/public/monitor/nodes", "/api/v1/public/monitor/config", "/api/v1/public/monitor/cluster", "/status/monitor"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		panel.e.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("public resource %s = %d %s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), settings.ClusterKey) || strings.Contains(response.Body.String(), "panelpass123") || strings.Contains(response.Body.String(), settings.AdvertiseURL) {
			t.Fatalf("public resource %s exposes credentials or candidate addresses", path)
		}
	}
	for _, path := range []string{"/api/v1/caddy/stop", "/api/v1/panel/restart"} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Header.Set("Authorization", "Bearer "+settings.ClusterKey)
		response := httptest.NewRecorder()
		panel.e.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("probe credential accessed administration: %s = %d", path, response.Code)
		}
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/public/monitor/nodes?command=exec", nil)
	response = httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatal("public monitoring accepted undocumented query actions")
	}
	request = httptest.NewRequest(http.MethodDelete, "/api/v1/public/monitor/nodes/external-probe-123", nil)
	response = httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed && response.Code != http.StatusNotFound {
		t.Fatalf("public monitoring exposed a mutation: %d", response.Code)
	}
}
