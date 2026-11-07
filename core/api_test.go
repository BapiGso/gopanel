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

	"gopanel/core/config"
	"gopanel/core/httpapi"
	"gopanel/core/storage"
)

func apiTestPanel(t *testing.T) *Core {
	t.Helper()
	useTempCoreConfig(t, config.Config{
		Panel:  config.PanelConfig{Port: ":8443", Path: "login", Username: "paneluser", Password: "panelpass123"},
		WebDAV: config.WebDAVConfig{Enable: true, Username: "davuser", Password: "davpass123"},
	})
	panel := New()
	panel.setupRoutes()
	return panel
}

func apiRequest(panel *Core, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.SetBasicAuth("paneluser", "panelpass123")
	if body != "" {
		request.Header.Set(echoHeaderContentType, "application/json")
	}
	response := httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	return response
}

func TestAPIAuthenticationReturnsJSON(t *testing.T) {
	panel := apiTestPanel(t)
	request := httptest.NewRequest(http.MethodGet, httpapi.Prefix, nil)
	response := httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"code":"unauthorized"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	request.SetBasicAuth("paneluser", "incorrect")
	response = httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d", response.Code)
	}
	response = apiRequest(panel, http.MethodGet, httpapi.Prefix, "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"version":"v1"`) {
		t.Fatalf("basic auth response = %d %s", response.Code, response.Body.String())
	}
}

func TestAPIAcceptsExistingBrowserSession(t *testing.T) {
	panel := apiTestPanel(t)
	form := url.Values{"username": {"paneluser"}, "password": {"panelpass123"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set(echoHeaderContentType, "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	panel.e.ServeHTTP(loginResponse, request)
	if loginResponse.Code != http.StatusFound {
		t.Fatalf("login failed: %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, httpapi.Prefix, nil)
	for _, cookie := range loginResponse.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session auth response = %d %s", response.Code, response.Body.String())
	}
}

func TestAPIHasNoLegacyActionDispatch(t *testing.T) {
	panel := apiTestPanel(t)
	for _, example := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodPost, "/api/v1/caddy/stop?status=restart", "", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/caddy/unknown", "", http.StatusNotFound},
		{http.MethodPatch, "/api/v1/webdav/settings", `{"option":{"enable":false}}`, http.StatusBadRequest},
		{http.MethodGet, "/api/v1/webdav/settings?param=disable", "", http.StatusBadRequest},
	} {
		response := apiRequest(panel, example.method, example.path, example.body)
		if response.Code != example.status || !strings.Contains(response.Body.String(), `"error":`) {
			t.Fatalf("%s %s = %d %s", example.method, example.path, response.Code, response.Body.String())
		}
	}
	if !config.Snapshot().WebDAV.Enable {
		t.Fatal("invalid requests changed WebDAV state")
	}
}

func TestAPISettingsPatchPreservesOmittedFieldsAndFalse(t *testing.T) {
	panel := apiTestPanel(t)
	for attempt := 0; attempt < 2; attempt++ {
		body, _ := json.Marshal(map[string]any{"enable": false, "revision": config.Current().Revision})
		response := apiRequest(panel, http.MethodPatch, "/api/v1/webdav/settings", string(body))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"enable":false`) {
			t.Fatalf("patch response = %d %s", response.Code, response.Body.String())
		}
	}
	settings := config.Snapshot().WebDAV
	if settings.Enable || settings.Username != "davuser" || settings.Password != "davpass123" {
		t.Fatalf("patch overwrote unrelated settings: enable=%t username=%s", settings.Enable, settings.Username)
	}
	response := apiRequest(panel, http.MethodGet, "/api/v1/panel/settings", "")
	if strings.Contains(response.Body.String(), "panelpass123") {
		t.Fatal("settings response exposes the panel password")
	}
}

func TestOpenAPIDescribesRegisteredResources(t *testing.T) {
	panel := apiTestPanel(t)
	response := apiRequest(panel, http.MethodGet, "/api/v1/openapi.json", "")
	if response.Code != http.StatusOK {
		t.Fatalf("OpenAPI response = %d %s", response.Code, response.Body.String())
	}
	var document struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("version = %q", document.OpenAPI)
	}
	for _, path := range []string{"/caddy/config", "/frp/{service}/config", "/files/{path}", "/cron/jobs/{name}", "/docker/containers/{id}/stop", "/terminals/{id}", "/firewall/rules/{id}", "/monitor/status", "/headscale/config"} {
		if len(document.Paths[path]) == 0 {
			t.Errorf("missing resource: %s", path)
		}
	}
	identifiers := map[string]bool{}
	for path, methods := range document.Paths {
		if !strings.HasPrefix(path, "/") {
			t.Errorf("invalid OpenAPI path: %q", path)
		}
		for _, operation := range methods {
			if operation.OperationID == "" || identifiers[operation.OperationID] {
				t.Errorf("empty or duplicated operation ID: %s", operation.OperationID)
			}
			identifiers[operation.OperationID] = true
		}
	}
}

func TestAPIPanelSettingsReportsPendingRestartAcrossRetries(t *testing.T) {
	panel := apiTestPanel(t)
	for attempt := 0; attempt < 2; attempt++ {
		body, _ := json.Marshal(map[string]any{"port": ":9443", "revision": config.Current().Revision})
		response := apiRequest(panel, http.MethodPatch, "/api/v1/panel/settings", string(body))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"restart_required":true`) {
			t.Fatalf("patch response = %d %s", response.Code, response.Body.String())
		}
	}
	response := apiRequest(panel, http.MethodGet, "/api/v1/panel/settings", "")
	if !strings.Contains(response.Body.String(), `"restart_required":true`) {
		t.Fatalf("read lost pending restart state: %s", response.Body.String())
	}
}

func TestAPIRejectsCrossOriginMutations(t *testing.T) {
	panel := apiTestPanel(t)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/webdav/settings", strings.NewReader(`{"enable":false}`))
	request.Header.Set(echoHeaderContentType, "application/json")
	request.Header.Set("Origin", "https://untrusted.example")
	request.SetBasicAuth("paneluser", "panelpass123")
	response := httptest.NewRecorder()
	panel.e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !config.Snapshot().WebDAV.Enable {
		t.Fatalf("cross-origin mutation was accepted: %d %s", response.Code, response.Body.String())
	}
}

func TestLegacyMutationRoutesNoLongerDispatchActions(t *testing.T) {
	panel := apiTestPanel(t)
	form := url.Values{"username": {"paneluser"}, "password": {"panelpass123"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set(echoHeaderContentType, "application/x-www-form-urlencoded")
	login := httptest.NewRecorder()
	panel.e.ServeHTTP(login, request)
	if login.Code != http.StatusFound {
		t.Fatal("login failed")
	}
	for _, example := range []struct{ method, path string }{
		{http.MethodPost, "/admin/website?status=enable"},
		{http.MethodPut, "/admin/frp?type=frps"},
		{http.MethodPost, "/admin/cron"},
	} {
		request := httptest.NewRequest(example.method, example.path, nil)
		for _, cookie := range login.Result().Cookies() {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		panel.e.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed && response.Code != http.StatusNotFound {
			t.Fatalf("%s %s returned %d %s", example.method, example.path, response.Code, response.Body.String())
		}
	}
}

func apiResponseRevision(t *testing.T, response *httptest.ResponseRecorder, field string) string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	var revision string
	if err := json.Unmarshal(payload.Data[field], &revision); err != nil || len(revision) != 64 {
		t.Fatalf("missing or invalid %s: %v", field, err)
	}
	return revision
}

func TestAPISettingsRequireRevisionWithoutChangingState(t *testing.T) {
	panel := apiTestPanel(t)
	initial := config.Current()
	for _, example := range []struct{ path, body string }{
		{"/panel/settings", `{"port":":9443"}`},
		{"/webdav/settings", `{"enable":false}`},
		{"/caddy/settings", `{"autostart":true}`},
		{"/frp/frps/settings", `{"autostart":true}`},
		{"/frp/frpc/settings", `{"autostart":true}`},
	} {
		response := apiRequest(panel, http.MethodPatch, httpapi.Prefix+example.path, example.body)
		if response.Code != http.StatusUnprocessableEntity || config.Current() != initial {
			t.Fatalf("missing revision was accepted: %s = %d %s", example.path, response.Code, response.Body.String())
		}
	}
}

func TestAPISettingsShareRevisionAndRejectStaleWrites(t *testing.T) {
	panel := apiTestPanel(t)
	initial := config.Current().Revision
	for _, resource := range []struct{ path, field string }{
		{"/panel/settings", "revision"},
		{"/webdav/settings", "revision"},
		{"/caddy/status", "settings_revision"},
		{"/frp/frps/status", "settings_revision"},
		{"/frp/frpc/status", "settings_revision"},
	} {
		response := apiRequest(panel, http.MethodGet, httpapi.Prefix+resource.path, "")
		if revision := apiResponseRevision(t, response, resource.field); revision != initial {
			t.Fatalf("%s returned a different settings revision", resource.path)
		}
	}
	body, _ := json.Marshal(map[string]any{"enable": false, "revision": initial})
	response := apiRequest(panel, http.MethodPatch, "/api/v1/webdav/settings", string(body))
	latest := apiResponseRevision(t, response, "revision")
	if latest == initial {
		t.Fatal("a committed update did not advance the revision")
	}
	body, _ = json.Marshal(map[string]any{"autostart": true, "revision": initial})
	response = apiRequest(panel, http.MethodPatch, "/api/v1/caddy/settings", string(body))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"revision_conflict"`) || config.Snapshot().Enable.Caddy {
		t.Fatalf("stale settings write = %d %s", response.Code, response.Body.String())
	}
	for _, path := range []string{"/caddy/settings", "/frp/frps/settings", "/frp/frpc/settings"} {
		body, _ = json.Marshal(map[string]any{"autostart": true, "revision": latest})
		response = apiRequest(panel, http.MethodPatch, httpapi.Prefix+path, string(body))
		latest = apiResponseRevision(t, response, "settings_revision")
		if !strings.Contains(response.Body.String(), `"state":"stopped"`) || !strings.Contains(response.Body.String(), `"autostart":true`) {
			t.Fatalf("saving autostart changed runtime state: %d %s", response.Code, response.Body.String())
		}
	}
}

func TestAPIConfigValidationPreservesDiskAndSnapshot(t *testing.T) {
	panel := apiTestPanel(t)
	initial := config.Current()
	path := filepath.Join(config.Directory(), config.FileName)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"port": ":70000", "revision": initial.Revision})
	response := apiRequest(panel, http.MethodPatch, "/api/v1/panel/settings", string(body))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"field":"port"`) {
		t.Fatalf("invalid configuration = %d %s", response.Code, response.Body.String())
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != string(original) || config.Current() != initial {
		t.Fatal("invalid request changed persisted or cached settings")
	}
}

func TestAPIExternalSettingsEditRequiresReadAndMerge(t *testing.T) {
	panel := apiTestPanel(t)
	initial := config.Current()
	external := initial.Config
	external.WebDAV.Username = "external-user"
	content, err := json.Marshal(external)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.AtomicWrite(filepath.Join(config.Directory(), config.FileName), content); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"enable": false, "revision": initial.Revision})
	response := apiRequest(panel, http.MethodPatch, "/api/v1/webdav/settings", string(body))
	if response.Code != http.StatusConflict || config.Current() != initial {
		t.Fatal("an unobserved external edit was overwritten")
	}
	response = apiRequest(panel, http.MethodGet, "/api/v1/webdav/settings", "")
	latest := apiResponseRevision(t, response, "revision")
	if latest == initial.Revision || !strings.Contains(response.Body.String(), `"username":"external-user"`) {
		t.Fatal("GET did not refresh externally edited settings")
	}
	body, _ = json.Marshal(map[string]any{"enable": false, "revision": latest})
	response = apiRequest(panel, http.MethodPatch, "/api/v1/webdav/settings", string(body))
	if response.Code != http.StatusOK || config.Snapshot().WebDAV.Username != "external-user" || config.Snapshot().WebDAV.Enable {
		t.Fatal("merged update did not preserve the external edit")
	}
}

func TestSettingsPagesRenderTypedConfigurationAndRevision(t *testing.T) {
	panel := apiTestPanel(t)
	form := url.Values{"username": {"paneluser"}, "password": {"panelpass123"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set(echoHeaderContentType, "application/x-www-form-urlencoded")
	login := httptest.NewRecorder()
	panel.e.ServeHTTP(login, request)
	if login.Code != http.StatusFound {
		t.Fatal("login failed")
	}
	for _, path := range []string{"/admin/security", "/admin/webdav", "/admin/website", "/admin/frp"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		for _, cookie := range login.Result().Cookies() {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		panel.e.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), config.Current().Revision) {
			t.Fatalf("settings page %s did not render its revision: %d", path, response.Code)
		}
		if strings.Contains(response.Body.String(), "getPanelConfig") {
			t.Fatalf("settings page %s still contains string-key configuration access", path)
		}
	}
}
