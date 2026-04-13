package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"gopanel/core/config"
)

func TestAdminRouteRequiresAuthenticatedSession(t *testing.T) {
	useTempCoreConfig(t, config.Config{
		Panel: config.PanelConfig{
			Port:     ":8443",
			Path:     "login",
			Username: "paneluser",
			Password: "panelpass123",
		},
		WebDAV: config.WebDAVConfig{
			Enable:   true,
			Username: "davuser",
			Password: "davpass",
		},
	})

	c := New()
	c.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/admin/monitor", nil)
	rec := httptest.NewRecorder()

	c.e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLoginSetsSecureSessionCookie(t *testing.T) {
	useTempCoreConfig(t, config.Config{
		Panel: config.PanelConfig{
			Port:     ":8443",
			Path:     "login",
			Username: "paneluser",
			Password: "panelpass123",
		},
	})

	c := New()
	c.setupRoutes()

	form := url.Values{
		"username": {"paneluser"},
		"password": {"panelpass123"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echoHeaderContentType, "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	c.e.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "panel_token" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("panel_token cookie not found")
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("panel_token should be HttpOnly")
	}
	if !sessionCookie.Secure {
		t.Fatal("panel_token should be Secure")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("panel_token SameSite = %v, want %v", sessionCookie.SameSite, http.SameSiteLaxMode)
	}
}

func TestAdminRouteAllowsAuthenticatedSession(t *testing.T) {
	useTempCoreConfig(t, config.Config{
		Panel: config.PanelConfig{
			Port:     ":8443",
			Path:     "login",
			Username: "paneluser",
			Password: "panelpass123",
		},
	})

	c := New()
	c.setupRoutes()

	form := url.Values{
		"username": {"paneluser"},
		"password": {"panelpass123"},
	}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set(echoHeaderContentType, "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	c.e.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusFound {
		t.Fatalf("login status = %d, want %d", loginRec.Code, http.StatusFound)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/admin/monitor", nil)
	for _, cookie := range loginRec.Result().Cookies() {
		adminReq.AddCookie(cookie)
	}
	adminRec := httptest.NewRecorder()

	c.e.ServeHTTP(adminRec, adminReq)

	if adminRec.Code == http.StatusUnauthorized {
		t.Fatal("authenticated request should not be unauthorized")
	}
}

func TestWebDAVRouteRequiresAuthentication(t *testing.T) {
	useTempCoreConfig(t, config.Config{
		Panel: config.PanelConfig{
			Port:     ":8443",
			Path:     "login",
			Username: "paneluser",
			Password: "panelpass123",
		},
		WebDAV: config.WebDAVConfig{
			Enable:   true,
			Username: "davuser",
			Password: "davpass",
		},
	})

	c := New()
	c.setupRoutes()

	req := httptest.NewRequest("PROPFIND", "/webdav/", nil)
	rec := httptest.NewRecorder()

	c.e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHTTPErrorHandlerPreservesStatusCode(t *testing.T) {
	useTempCoreConfig(t, config.Config{
		Panel: config.PanelConfig{
			Port:     ":8443",
			Path:     "login",
			Username: "paneluser",
			Password: "panelpass123",
		},
	})

	c := New()
	c.setupRoutes()
	c.e.GET("/test-403", func(ctx *echo.Context) error {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-403", nil)
	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

const echoHeaderContentType = "Content-Type"

func useTempCoreConfig(t *testing.T, initial config.Config) {
	t.Helper()

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
	if err := os.WriteFile(path, mustMarshalConfig(t, initial), 0644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init() error = %v", err)
	}
}

func mustMarshalConfig(t *testing.T, cfg config.Config) []byte {
	t.Helper()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	return data
}
