package webdav

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
)

func TestFileSystemRejectsUnauthenticatedRequests(t *testing.T) {
	useTempWebDAVConfig(t, config.Config{
		WebDAV: config.WebDAVConfig{
			Enable:   true,
			Username: "davuser",
			Password: "davpass",
		},
	})

	e := echo.New()
	req := httptest.NewRequest("PROPFIND", "/webdav/", nil)
	rec := httptest.NewRecorder()

	FileSystem()(e.NewContext(req, rec))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatal("WWW-Authenticate header is empty")
	}
}

func TestFileSystemRejectsInvalidCredentials(t *testing.T) {
	useTempWebDAVConfig(t, config.Config{
		WebDAV: config.WebDAVConfig{
			Enable:   true,
			Username: "davuser",
			Password: "davpass",
		},
	})

	e := echo.New()
	req := httptest.NewRequest("PROPFIND", "/webdav/", nil)
	req.Header.Set("Authorization", basicAuth("davuser", "wrongpass"))
	rec := httptest.NewRecorder()

	FileSystem()(e.NewContext(req, rec))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestFileSystemAllowsAuthenticatedPropfindWhenEnabled(t *testing.T) {
	useTempWebDAVConfig(t, config.Config{
		WebDAV: config.WebDAVConfig{
			Enable:   true,
			Username: "davuser",
			Password: "davpass",
		},
	})

	e := echo.New()
	req := httptest.NewRequest("PROPFIND", "/webdav/", nil)
	req.Header.Set("Authorization", basicAuth("davuser", "davpass"))
	req.Header.Set("Depth", "0")
	rec := httptest.NewRecorder()

	FileSystem()(e.NewContext(req, rec))

	if rec.Code != http.StatusMultiStatus {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMultiStatus)
	}
}

func TestFileSystemRejectsWhenDisabled(t *testing.T) {
	useTempWebDAVConfig(t, config.Config{
		WebDAV: config.WebDAVConfig{
			Enable:   false,
			Username: "davuser",
			Password: "davpass",
		},
	})

	e := echo.New()
	req := httptest.NewRequest("PROPFIND", "/webdav/", nil)
	req.Header.Set("Authorization", basicAuth("davuser", "davpass"))
	rec := httptest.NewRecorder()

	FileSystem()(e.NewContext(req, rec))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func useTempWebDAVConfig(t *testing.T, initial config.Config) string {
	t.Helper()

	initial.Panel = config.PanelConfig{
		Port:     ":8443",
		Path:     "panelpath",
		Username: "paneluser",
		Password: "panelpass",
	}

	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
		if err := config.Init(); err != nil {
			t.Fatalf("reload original config: %v", err)
		}
	})

	path := filepath.Join(tempDir, "gopanel_config.json")
	data, err := json.MarshalIndent(initial, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init() error = %v", err)
	}

	return path
}

func basicAuth(username, password string) string {
	token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return "Basic " + token
}
