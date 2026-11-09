package mymiddleware

import (
	"bytes"
	"encoding/json"
	"html"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestRealRendererExecutesEveryPageWithEscapedBootstrap(t *testing.T) {
	hostile := `quotes'"` + "`" + `</script><img src=x onerror=attack()>&`
	pages := []struct {
		name string
		data any
	}{
		{"login", nil},
		{"monitor", map[string]any{"title": hostile, "api_base": "/api/v1/monitor", "public": false}},
		{"monitor", map[string]any{"title": hostile, "api_base": "/api/v1/public/monitor", "public": true}},
		{"website", map[string]any{"caddyFile": hostile, "caddyRevision": "revision", "caddyEnable": true, "settingsRevision": "settings"}},
		{"file", map[string]any{"Dir": "/" + hostile, "Files": []map[string]any{{"name": hostile, "path": "/" + hostile, "type": "file", "size": 12, "permissions": "0644", "modified_at": "2026-01-01T00:00:00Z"}}}},
		{"webdav", map[string]any{"webdavConfig": map[string]any{"Username": hostile, "Enable": true}, "settingsRevision": "settings"}},
		{"term", []map[string]any{{"id": "session", "rows": 24, "cols": 80}}},
		{"security", map[string]any{"panelConfig": map[string]any{"Username": hostile, "Port": ":8443", "Path": hostile}, "settingsRevision": "settings", "version": "Development build"}},
		{"cron", []map[string]any{{"name": hostile, "script": hostile, "schedule": "0 0 * * *", "paused": false}}},
		{"docker", nil},
		{"frp", map[string]any{"frpsConfig": hostile, "frpcConfig": hostile, "frpsRevision": "server", "frpcRevision": "client", "frpsEnable": false, "frpcEnable": true, "settingsRevision": "settings"}},
		{"headscale", map[string]any{"headscaleEnable": false}},
		{"firewall", []map[string]any{{"id": hostile, "network": "ipv4", "protocol": "tcp", "port": 80, "direction": "in", "action": "accept"}}},
		{"warning", map[string]any{"message": hostile, "ip": hostile}},
		{"unavailable", nil},
	}
	bootstrap := regexp.MustCompile(`data-page="([^"]*)"`)
	for _, page := range pages {
		t.Run(page.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := DefaultTemplateRender.Render(nil, &output, page.name+".template", page.data); err != nil {
				t.Fatal(err)
			}
			markup := output.String()
			if strings.Contains(markup, "ZgotmplZ") || strings.Contains(markup, hostile) {
				t.Fatal("renderer emitted an unsafe value")
			}
			if !strings.Contains(markup, `name="viewport" content="width=device-width, initial-scale=1"`) || !strings.Contains(markup, "<cds-") {
				t.Fatal("renderer omitted shared responsive Carbon shell")
			}
			matches := bootstrap.FindAllStringSubmatch(markup, -1)
			if len(matches) != 1 {
				t.Fatalf("expected one page bootstrap, got %d", len(matches))
			}
			var decoded any
			if err := json.Unmarshal([]byte(html.UnescapeString(matches[0][1])), &decoded); err != nil {
				t.Fatalf("page bootstrap is not valid JSON: %v", err)
			}
			data := page.data
			if data == nil {
				data = map[string]any{}
			}
			encoded, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			var expected any
			if err := json.Unmarshal(encoded, &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded, expected) {
				t.Fatalf("page data did not round-trip: got %v, want %v", decoded, expected)
			}
			if page.name == "monitor" {
				public := page.data.(map[string]any)["public"].(bool)
				if strings.Contains(markup, "<cds-side-nav ") == public {
					t.Fatal("public and admin monitor shells were not separated")
				}
			}
		})
	}
}

func TestIconOnlyAcceptsEmbeddedIconsAndSafeSizes(t *testing.T) {
	for _, request := range []struct {
		name string
		size int
	}{{"../security", 32}, {"security", 0}, {"security", 65}, {"missing", 32}} {
		if _, err := icon(request.name, request.size); err == nil {
			t.Fatalf("accepted invalid icon request %+v", request)
		}
	}
	markup, err := icon("sun", 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, attribute := range []string{`class="ui-icon"`, `aria-hidden="true"`, `focusable="false"`, `width="20"`, `height="20"`} {
		if !strings.Contains(string(markup), attribute) {
			t.Errorf("icon omitted %s", attribute)
		}
	}
}
