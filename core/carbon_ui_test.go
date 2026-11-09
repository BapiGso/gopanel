package core

import (
	"bytes"
	"encoding/json"
	"html"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gopanel/assets"
	"gopanel/core/config"
	"gopanel/core/mymiddleware"
)

func TestCarbonTemplateIconsHaveProductiveSizesAndNoTitleIcons(t *testing.T) {
	templates, err := fs.Glob(assets.Assets, "*.template")
	if err != nil {
		t.Fatal(err)
	}
	iconCall := regexp.MustCompile(`\{\{\s*icon\s+"[^"]+"\s+(\d+)\s*\}\}`)
	headings := regexp.MustCompile(`(?s)<h1\b[^>]*>(.*?)</h1>`)
	anyIcon := regexp.MustCompile(`\{\{-?\s*icon\b`)
	for _, name := range templates {
		content, err := assets.Assets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, heading := range headings.FindAllSubmatch(content, -1) {
			if anyIcon.Match(heading[1]) {
				t.Errorf("page title in %s contains an icon", name)
			}
		}
		for _, match := range iconCall.FindAllSubmatch(content, -1) {
			size, err := strconv.Atoi(string(match[1]))
			if err != nil {
				t.Fatal(err)
			}
			emptyState := name == "warning.template" || name == "unavailable.template"
			header := name == "header.template" || name == "public-header.template"
			allowed := size == 16 || header && size == 20 || emptyState && size > 20 && size <= 64
			if !allowed {
				t.Errorf("unexpected icon size %d in %s", size, name)
			}
		}
	}
}

func TestCarbonRenderedIconsHaveExactDimensionsAndScopedClass(t *testing.T) {
	for _, example := range []struct {
		page  string
		sizes []int
	}{{"website.template", []int{20, 20, 16}}, {"warning.template", []int{20, 20, 64}}} {
		var output bytes.Buffer
		if err := mymiddleware.DefaultTemplateRender.Render(nil, &output, example.page, nil); err != nil {
			t.Fatal(err)
		}
		icons := regexp.MustCompile(`<svg\b[^>]*>`).FindAllString(output.String(), -1)
		if len(icons) != len(example.sizes) {
			t.Fatalf("%s rendered %d icons, want %d", example.page, len(icons), len(example.sizes))
		}
		for index, icon := range icons {
			size := strconv.Itoa(example.sizes[index])
			for _, attribute := range []string{`class="ui-icon"`, `width="` + size + `"`, `height="` + size + `"`, `aria-hidden="true"`, `focusable="false"`} {
				if !strings.Contains(icon, attribute) {
					t.Errorf("%s icon omitted exact %s", example.page, attribute)
				}
			}
		}
	}
}

func TestSettingsPagesDoNotBootstrapPasswords(t *testing.T) {
	panel := apiTestPanel(t)
	settings := config.Snapshot()
	form := url.Values{"username": {settings.Panel.Username}, "password": {settings.Panel.Password}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set(echoHeaderContentType, "application/x-www-form-urlencoded")
	login := httptest.NewRecorder()
	panel.e.ServeHTTP(login, request)
	if login.Code != http.StatusFound {
		t.Fatalf("login returned %d", login.Code)
	}
	bootstrap := regexp.MustCompile(`data-page="([^"]*)"`)
	for _, path := range []string{"/admin/security", "/admin/webdav"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		for _, cookie := range login.Result().Cookies() {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		panel.e.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d: %s", path, response.Code, response.Body.String())
		}
		markup := response.Body.String()
		if strings.Contains(markup, settings.Panel.Password) || strings.Contains(markup, settings.WebDAV.Password) {
			t.Fatalf("%s exposed an existing password", path)
		}
		matches := bootstrap.FindStringSubmatch(markup)
		if len(matches) != 2 {
			t.Fatalf("%s omitted page bootstrap", path)
		}
		data := html.UnescapeString(matches[1])
		if !json.Valid([]byte(data)) || strings.Contains(strings.ToLower(data), "password") {
			t.Fatalf("%s bootstrapped secret configuration fields", path)
		}
	}
}

func TestUISettingsWritesOmitPasswordsAndPreserveThem(t *testing.T) {
	panel := apiTestPanel(t)
	before := config.Snapshot()
	for _, example := range []struct {
		path string
		body map[string]any
	}{
		{"/api/v1/panel/settings", map[string]any{"username": before.Panel.Username, "port": before.Panel.Port, "path": before.Panel.Path}},
		{"/api/v1/webdav/settings", map[string]any{"username": before.WebDAV.Username, "enable": false}},
	} {
		example.body["revision"] = config.Current().Revision
		body, err := json.Marshal(example.body)
		if err != nil {
			t.Fatal(err)
		}
		response := apiRequest(panel, http.MethodPatch, example.path, string(body))
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d: %s", example.path, response.Code, response.Body.String())
		}
		if after := config.Snapshot(); after.Panel.Password != before.Panel.Password || after.WebDAV.Password != before.WebDAV.Password {
			t.Fatal("an omitted UI password replaced the stored credential")
		}
	}
}

func TestVendorCachingAndPWARewritesRemainAvailable(t *testing.T) {
	panel := apiTestPanel(t)
	for _, path := range []string{"/assets/vendor/carbon/v2.67.0/button.min.js", "/assets/vendor/carbon-charts/1.27.27/styles.min.css", "/assets/vendor/carbon/fonts/IBMPlexSans-Regular-Latin1.woff2", "/assets/vendor/jsvectormap/jsvectormap.min.js", "/manifest.webmanifest", "/sw.js", "/assets/js/api.js"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Accept-Encoding", "gzip")
		response := httptest.NewRecorder()
		panel.e.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, response.Code)
		}
		cache := response.Header().Get("Cache-Control")
		versioned := strings.HasPrefix(path, "/assets/vendor/carbon/v2.67.0/") || strings.HasPrefix(path, "/assets/vendor/carbon-charts/1.27.27/")
		if versioned && cache != "public, max-age=31536000, immutable" {
			t.Errorf("%s cache policy = %q", path, cache)
		}
		if !versioned && strings.Contains(cache, "immutable") {
			t.Errorf("unversioned asset %s received an immutable cache policy", path)
		}
		if path == "/assets/vendor/carbon/v2.67.0/button.min.js" && response.Header().Get("Content-Encoding") != "gzip" {
			t.Fatal("static gzip behaviour was removed")
		}
	}
}
