package website

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/httpapi"
	"gopanel/core/storage"
)

func testWebsiteAPI(t *testing.T, content string) *echo.Echo {
	t.Helper()
	if err := config.Init(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	oldFile := caddyFile
	caddyFile = &storage.File{Path: filepath.Join(t.TempDir(), "Caddyfile")}
	if err := os.WriteFile(caddyFile.Path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { caddyFile = oldFile })
	server := echo.New()
	server.HTTPErrorHandler = httpapi.ErrorHandler
	RegisterAPI(httpapi.NewRouter(server.Group(httpapi.Prefix)))
	return server
}

func TestSaveCaddyConfigDoesNotApplyAndRejectsStaleRevision(t *testing.T) {
	server := testWebsiteAPI(t, "{\nadmin off\n}\n:2015\nrespond old\n")
	initial, err := caddyFile.Read()
	if err != nil {
		t.Fatal(err)
	}
	content := "{\nadmin off\n}\n:2015\nrespond updated\n"
	body, _ := json.Marshal(map[string]any{"content": content, "revision": initial.Revision})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/caddy/config", strings.NewReader(string(body)))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save failed: %d %s", response.Code, response.Body.String())
	}
	if caddyRunning {
		t.Fatal("saving configuration started Caddy")
	}
	current, _ := caddyFile.Read()
	if current.Content != content {
		t.Fatalf("saved content = %q", current.Content)
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, "/api/v1/caddy/config", strings.NewReader(string(body)))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("stale write status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestEmbeddedCaddyDisablesNativeAdminAndAutosave(t *testing.T) {
	encoded, err := adaptConfig("{\nadmin localhost:2019\n}\n:2015\nrespond hello\n")
	if err != nil {
		t.Fatal(err)
	}
	var configuration caddy.Config
	if err := json.Unmarshal(encoded, &configuration); err != nil {
		t.Fatal(err)
	}
	if configuration.Admin == nil || !configuration.Admin.Disabled || configuration.Admin.Config == nil || configuration.Admin.Config.Persist == nil || *configuration.Admin.Config.Persist {
		t.Fatal("embedded admin API or autosave remains enabled")
	}
}

func TestHTTPStopsTheSameEmbeddedCaddyInstance(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	server := testWebsiteAPI(t, "{\nadmin off\nauto_https off\n}\nhttp://"+address+"\nrespond panel-instance\n")
	t.Cleanup(func() { _ = caddyStop() })
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/caddy/start", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", response.Code, response.Body.String())
	}
	client := &http.Client{Timeout: time.Second}
	served, err := client.Get("http://" + address)
	if err != nil {
		t.Fatal(err)
	}
	served.Body.Close()
	response = httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/caddy/stop", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"state":"stopped"`) {
		t.Fatalf("stop failed: %d %s", response.Code, response.Body.String())
	}
	connection, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("HTTP stop did not release the embedded Caddy listener")
	}
}
