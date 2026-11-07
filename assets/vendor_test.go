package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestVendoredWorldMapIntegrity(t *testing.T) {
	content, err := Assets.ReadFile("vendor/jsvectormap/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Version string            `json:"version"`
		Files   map[string]string `json:"files"`
	}
	if err := json.Unmarshal(content, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Version != "1.7.0" || len(metadata.Files) != 3 {
		t.Fatal("unexpected map asset manifest")
	}
	for name, expected := range metadata.Files {
		content, err := Assets.ReadFile("vendor/jsvectormap/" + name)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(content)
		if hex.EncodeToString(hash[:]) != expected {
			t.Fatalf("vendored asset %s differs from its verified distribution", name)
		}
	}
}

func TestMonitorStylesArePrecompiledAndBounded(t *testing.T) {
	content, err := Assets.ReadFile("css/observatory.min.css")
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 || len(content) > 128<<10 {
		t.Fatalf("monitor CSS is empty or includes an unnecessarily large full framework: %d bytes", len(content))
	}
	for _, theme := range []string{"data-theme=business", "data-theme=winter"} {
		if !strings.Contains(string(content), theme) {
			t.Fatalf("compiled CSS is missing theme %s", theme)
		}
	}
}
