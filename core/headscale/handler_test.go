////go:build linux || darwin || freebsd

package headscale

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

// ==================== loadServerConfig 测试 ====================

func TestLoadServerConfig_ValidConfig(t *testing.T) {
	cfg := &headscaleConfig{
		ServerURL:         "https://myheadscale.example.com:443",
		ListenAddr:        "0.0.0.0:8080",
		MetricsListenAddr: "127.0.0.1:9090",
		PrivateKeyPath:    "/var/lib/headscale/noise_private.key",
		IPv4Prefix:        "100.64.0.0/10",
		IPv6Prefix:        "fd7a:115c:a1e0::/48",
		BaseDomain:        "example.com",
	}

	result, err := loadServerConfig(cfg)
	if err != nil {
		t.Fatalf("loadServerConfig failed with valid config: %v", err)
	}

	if result.ServerURL != cfg.ServerURL {
		t.Errorf("ServerURL = %q, want %q", result.ServerURL, cfg.ServerURL)
	}
	if result.Addr != cfg.ListenAddr {
		t.Errorf("Addr = %q, want %q", result.Addr, cfg.ListenAddr)
	}
	if result.MetricsAddr != cfg.MetricsListenAddr {
		t.Errorf("MetricsAddr = %q, want %q", result.MetricsAddr, cfg.MetricsListenAddr)
	}
	if result.GRPCAddr != "127.0.0.1:50443" {
		t.Errorf("GRPCAddr = %q, want %q", result.GRPCAddr, "127.0.0.1:50443")
	}
	if result.NoisePrivateKeyPath != cfg.PrivateKeyPath {
		t.Errorf("NoisePrivateKeyPath = %q, want %q", result.NoisePrivateKeyPath, cfg.PrivateKeyPath)
	}
	if result.BaseDomain != cfg.BaseDomain {
		t.Errorf("BaseDomain = %q, want %q", result.BaseDomain, cfg.BaseDomain)
	}
	if result.PrefixV4 == nil {
		t.Error("PrefixV4 is nil")
	}
	if result.PrefixV6 == nil {
		t.Error("PrefixV6 is nil")
	}
	if result.Database.Type != "sqlite3" {
		t.Errorf("Database.Type = %q, want %q", result.Database.Type, "sqlite3")
	}
	if !result.Database.Sqlite.WriteAheadLog {
		t.Error("WriteAheadLog should be enabled")
	}
	if result.GRPCAllowInsecure != true {
		t.Error("GRPCAllowInsecure should be true")
	}
	if len(result.DERP.URLs) != 1 {
		t.Errorf("DERP.URLs length = %d, want 1", len(result.DERP.URLs))
	}
}

func TestLoadServerConfig_InvalidIPv4Prefix(t *testing.T) {
	cfg := &headscaleConfig{
		ServerURL:         "https://example.com",
		ListenAddr:        "0.0.0.0:8080",
		MetricsListenAddr: "127.0.0.1:9090",
		PrivateKeyPath:    "/tmp/key",
		IPv4Prefix:        "invalid-prefix",
		IPv6Prefix:        "fd7a:115c:a1e0::/48",
		BaseDomain:        "example.com",
	}

	_, err := loadServerConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid IPv4 prefix, got nil")
	}
	if !strings.Contains(err.Error(), "IPv4") {
		t.Errorf("error should mention IPv4, got: %v", err)
	}
}

func TestLoadServerConfig_InvalidIPv6Prefix(t *testing.T) {
	cfg := &headscaleConfig{
		ServerURL:         "https://example.com",
		ListenAddr:        "0.0.0.0:8080",
		MetricsListenAddr: "127.0.0.1:9090",
		PrivateKeyPath:    "/tmp/key",
		IPv4Prefix:        "100.64.0.0/10",
		IPv6Prefix:        "not-valid",
		BaseDomain:        "example.com",
	}

	_, err := loadServerConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid IPv6 prefix, got nil")
	}
	if !strings.Contains(err.Error(), "IPv6") {
		t.Errorf("error should mention IPv6, got: %v", err)
	}
}

// ==================== DERP 配置测试 ====================

func TestLoadServerConfig_DERPEnabled(t *testing.T) {
	cfg := &headscaleConfig{
		ServerURL:         "https://example.com",
		ListenAddr:        "0.0.0.0:8080",
		MetricsListenAddr: "127.0.0.1:9090",
		PrivateKeyPath:    "/tmp/key",
		IPv4Prefix:        "100.64.0.0/10",
		IPv6Prefix:        "fd7a:115c:a1e0::/48",
		BaseDomain:        "example.com",
		DERPEnabled:       true,
		DERPRegionID:      100,
		DERPRegionCode:    "my-region",
		DERPRegionName:    "My Custom DERP",
		DERPSTUNAddr:      "0.0.0.0:3479",
		DERPVerifyClient:  "on",
	}

	result, err := loadServerConfig(cfg)
	if err != nil {
		t.Fatalf("loadServerConfig failed: %v", err)
	}

	derp := result.DERP
	if !derp.ServerEnabled {
		t.Error("DERP.ServerEnabled should be true")
	}
	if !derp.AutomaticallyAddEmbeddedDerpRegion {
		t.Error("DERP.AutomaticallyAddEmbeddedDerpRegion should be true")
	}
	if derp.ServerRegionID != 100 {
		t.Errorf("DERP.ServerRegionID = %d, want 100", derp.ServerRegionID)
	}
	if derp.ServerRegionCode != "my-region" {
		t.Errorf("DERP.ServerRegionCode = %q, want %q", derp.ServerRegionCode, "my-region")
	}
	if derp.ServerRegionName != "My Custom DERP" {
		t.Errorf("DERP.ServerRegionName = %q, want %q", derp.ServerRegionName, "My Custom DERP")
	}
	if derp.STUNAddr != "0.0.0.0:3479" {
		t.Errorf("DERP.STUNAddr = %q, want %q", derp.STUNAddr, "0.0.0.0:3479")
	}
	if !derp.ServerVerifyClients {
		t.Error("DERP.ServerVerifyClients should be true")
	}
	if derp.ServerPrivateKeyPath == "" {
		t.Error("DERP.ServerPrivateKeyPath should not be empty")
	}
	if !strings.HasSuffix(derp.ServerPrivateKeyPath, "/derp_server.key") {
		t.Errorf("DERP.ServerPrivateKeyPath = %q, should end with /derp_server.key", derp.ServerPrivateKeyPath)
	}
	// 即使启用了内嵌 DERP，官方 URL 仍作为 fallback
	if len(derp.URLs) != 1 {
		t.Errorf("DERP.URLs length = %d, want 1 (fallback)", len(derp.URLs))
	}
}

func TestLoadServerConfig_DERPDisabled(t *testing.T) {
	cfg := &headscaleConfig{
		ServerURL:         "https://example.com",
		ListenAddr:        "0.0.0.0:8080",
		MetricsListenAddr: "127.0.0.1:9090",
		PrivateKeyPath:    "/tmp/key",
		IPv4Prefix:        "100.64.0.0/10",
		IPv6Prefix:        "fd7a:115c:a1e0::/48",
		BaseDomain:        "example.com",
		DERPEnabled:       false,
	}

	result, err := loadServerConfig(cfg)
	if err != nil {
		t.Fatalf("loadServerConfig failed: %v", err)
	}

	if result.DERP.ServerEnabled {
		t.Error("DERP.ServerEnabled should be false")
	}
	if result.DERP.AutomaticallyAddEmbeddedDerpRegion {
		t.Error("DERP.AutomaticallyAddEmbeddedDerpRegion should be false")
	}
}

func TestLoadServerConfig_DERPVerifyClientOff(t *testing.T) {
	cfg := &headscaleConfig{
		ServerURL:         "https://example.com",
		ListenAddr:        "0.0.0.0:8080",
		MetricsListenAddr: "127.0.0.1:9090",
		PrivateKeyPath:    "/tmp/key",
		IPv4Prefix:        "100.64.0.0/10",
		IPv6Prefix:        "fd7a:115c:a1e0::/48",
		BaseDomain:        "example.com",
		DERPEnabled:       true,
		DERPVerifyClient:  "",
	}

	result, err := loadServerConfig(cfg)
	if err != nil {
		t.Fatalf("loadServerConfig failed: %v", err)
	}

	if result.DERP.ServerVerifyClients {
		t.Error("DERP.ServerVerifyClients should be false when DERPVerifyClient is empty")
	}
}

// ==================== Handler API 测试 ====================

// helper: 创建 echo 上下文
func newTestContext(method, path string, form url.Values) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// helper: 解析 JSON 响应
func parseJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON response: %v, body: %s", err, rec.Body.String())
	}
	return result
}

func TestCheckStatus_WhenStopped(t *testing.T) {
	hsMutex.Lock()
	hsRunning = false
	hsError = ""
	hsMutex.Unlock()

	c, rec := newTestContext("POST", "/admin/headscale?status=check", nil)
	err := Index(c)
	if err != nil {
		t.Fatalf("Index returned error: %v", err)
	}

	result := parseJSON(t, rec)
	if result["status"] != "stopped" {
		t.Errorf("status = %v, want 'stopped'", result["status"])
	}
}

func TestCheckStatus_WhenRunning(t *testing.T) {
	hsMutex.Lock()
	hsRunning = true
	hsError = ""
	hsMutex.Unlock()

	c, rec := newTestContext("POST", "/admin/headscale?status=check", nil)
	err := Index(c)
	if err != nil {
		t.Fatalf("Index returned error: %v", err)
	}

	result := parseJSON(t, rec)
	if result["status"] != "running" {
		t.Errorf("status = %v, want 'running'", result["status"])
	}

	// 恢复状态
	hsMutex.Lock()
	hsRunning = false
	hsMutex.Unlock()
}

func TestStartAlreadyRunning(t *testing.T) {
	hsMutex.Lock()
	hsRunning = true
	hsMutex.Unlock()

	c, rec := newTestContext("POST", "/admin/headscale?status=start", nil)
	err := Index(c)
	if err != nil {
		t.Fatalf("Index returned error: %v", err)
	}

	result := parseJSON(t, rec)
	if result["success"] != true {
		t.Errorf("success = %v, want true", result["success"])
	}
	if result["message"] != "Already running" {
		t.Errorf("message = %v, want 'Already running'", result["message"])
	}

	hsMutex.Lock()
	hsRunning = false
	hsMutex.Unlock()
}

func TestStopWhenNotRunning(t *testing.T) {
	hsMutex.Lock()
	hsRunning = false
	hsMutex.Unlock()

	c, rec := newTestContext("POST", "/admin/headscale?status=stop", nil)
	err := Index(c)
	if err != nil {
		t.Fatalf("Index returned error: %v", err)
	}

	result := parseJSON(t, rec)
	if result["success"] != false {
		t.Errorf("success = %v, want false", result["success"])
	}
}

func TestCheckStatus_WithError(t *testing.T) {
	hsMutex.Lock()
	hsRunning = false
	hsError = "bind: address already in use"
	hsMutex.Unlock()

	c, rec := newTestContext("POST", "/admin/headscale?status=check", nil)
	err := Index(c)
	if err != nil {
		t.Fatalf("Index returned error: %v", err)
	}

	result := parseJSON(t, rec)
	if result["error"] != "bind: address already in use" {
		t.Errorf("error = %v, want error message", result["error"])
	}

	hsMutex.Lock()
	hsError = ""
	hsMutex.Unlock()
}

// ==================== 集成测试 ====================
// 需要在 Linux 上运行，并且需要合适的文件权限

func TestHeadscaleStartAndConnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 切换到 /tmp 避免 WSL 的 /mnt/c 不支持 Unix socket
	origDir, _ := os.Getwd()
	testDir := "/tmp/headscale_test_" + fmt.Sprint(os.Getpid())
	os.MkdirAll(testDir, 0755)
	os.Chdir(testDir)
	defer func() {
		os.Chdir(origDir)
		os.RemoveAll(testDir)
	}()

	// 准备表单数据（启用内嵌 DERP）
	form := url.Values{
		"server_url":          {"http://127.0.0.1:18080"},
		"listen_addr":         {"127.0.0.1:18080"},
		"metrics_listen_addr": {"127.0.0.1:19090"},
		"private_key_path":    {testDir + "/noise_private.key"},
		"ipv4_prefix":         {"100.64.0.0/10"},
		"ipv6_prefix":         {"fd7a:115c:a1e0::/48"},
		"base_domain":         {"test.example.com"},
		"derp_enabled":        {"true"},
		"derp_region_id":      {"999"},
		"derp_region_code":    {"gopanel"},
		"derp_region_name":    {"GoPanel Embedded DERP"},
		"derp_stun_addr":      {"0.0.0.0:3478"},
	}

	// 1. 启动 Headscale
	t.Log("Starting Headscale with embedded DERP...")
	c, rec := newTestContext("POST", "/admin/headscale?status=start", form)
	err := Index(c)
	if err != nil {
		t.Fatalf("start request failed: %v", err)
	}

	result := parseJSON(t, rec)
	t.Logf("Start response: %v", result)
	if result["success"] != true {
		t.Fatalf("start failed: %v", result["message"])
	}

	// 2. 等待服务启动，然后检查状态
	t.Log("Waiting for service to start...")
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)

		c2, rec2 := newTestContext("POST", "/admin/headscale?status=check", nil)
		_ = Index(c2)
		status := parseJSON(t, rec2)

		if status["status"] == "running" {
			t.Log("Service is running!")
			break
		}

		if errMsg, ok := status["error"].(string); ok && errMsg != "" {
			t.Fatalf("service failed to start: %s", errMsg)
		}

		t.Logf("Attempt %d: status=%v, waiting...", i+1, status["status"])
	}

	// 3. 等待端口绑定完成，然后尝试 HTTP 连接
	t.Log("Testing HTTP connection to Headscale...")
	var connected bool
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:18080/health")
		if err != nil {
			t.Logf("Attempt %d: connection failed: %v", i+1, err)
			continue
		}
		resp.Body.Close()
		t.Logf("HTTP response status: %d", resp.StatusCode)
		connected = true
		break
	}
	if !connected {
		t.Error("Failed to connect to Headscale after retries")
	} else {
		t.Log("Headscale HTTP connection verified!")
	}

	// 4. 停止服务
	t.Log("Stopping Headscale...")
	c3, rec3 := newTestContext("POST", "/admin/headscale?status=stop", nil)
	err = Index(c3)
	if err != nil {
		t.Fatalf("stop request failed: %v", err)
	}
	stopResult := parseJSON(t, rec3)
	t.Logf("Stop response: %v", stopResult)
}
