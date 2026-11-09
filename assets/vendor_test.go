package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
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

func TestCarbonManifestIntegrityAndUsage(t *testing.T) {
	content, err := Assets.ReadFile("vendor/carbon/v2.67.0/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version    string   `json:"version"`
		SourceURL  string   `json:"source_url"`
		Components []string `json:"components"`
		Icons      []string `json:"icons"`
		Files      map[string]struct {
			SHA256 string `json:"sha256"`
			Source string `json:"source"`
		} `json:"files"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "2.67.0" || !strings.HasSuffix(manifest.SourceURL, "/version/v2.67.0/") || len(manifest.Files) == 0 {
		t.Fatal("Carbon manifest is missing pinned distribution information")
	}
	for name, record := range manifest.Files {
		embedded, err := Assets.ReadFile(name)
		if err != nil {
			t.Fatalf("Carbon file %s is not embedded: %v", name, err)
		}
		disk, err := os.ReadFile(filepath.FromSlash(name))
		if err != nil {
			t.Fatalf("Carbon file %s does not exist on disk: %v", name, err)
		}
		for _, contents := range [][]byte{embedded, disk} {
			hash := sha256.Sum256(contents)
			if record.Source == "" || hex.EncodeToString(hash[:]) != record.SHA256 {
				t.Fatalf("Carbon file %s does not match its manifest", name)
			}
		}
	}
	componentPattern := regexp.MustCompile(`/assets/vendor/carbon/v2\.67\.0/([a-z-]+)\.min\.js`)
	iconPattern := regexp.MustCompile(`icon "([a-z0-9-]+)"`)
	components, icons := map[string]bool{}, map[string]bool{}
	templates, err := fs.Glob(Assets, "*.template")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range templates {
		content, err := Assets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range componentPattern.FindAllSubmatch(content, -1) {
			components[string(match[1])] = true
		}
		for _, match := range iconPattern.FindAllSubmatch(content, -1) {
			icons[string(match[1])] = true
		}
	}
	for _, collection := range []struct {
		name     string
		used     map[string]bool
		vendored []string
	}{{"component", components, manifest.Components}, {"icon", icons, manifest.Icons}} {
		if len(collection.used) != len(collection.vendored) {
			t.Errorf("vendored %s count %d differs from used count %d", collection.name, len(collection.vendored), len(collection.used))
		}
		for name := range collection.used {
			if !slices.Contains(collection.vendored, name) {
				t.Errorf("used Carbon %s %s is not vendored", collection.name, name)
			}
		}
	}
}

func TestCarbonImportGraphAndFontsAreLocal(t *testing.T) {
	imports := []*regexp.Regexp{regexp.MustCompile(`\bfrom\s*["']([^"']+)["']`), regexp.MustCompile(`\bimport\s*(?:\(\s*)?["']([^"']+)["']`)}
	remoteURL := regexp.MustCompile(`(?i)url\(\s*["']?(?:https?:)?//`)
	err := fs.WalkDir(Assets, "vendor/carbon", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !slices.Contains([]string{".js", ".css"}, path.Ext(name)) {
			return nil
		}
		content, err := Assets.ReadFile(name)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "sourceMappingURL=") || remoteURL.Match(content) {
			t.Errorf("Carbon asset %s references non-local runtime resources", name)
		}
		if path.Ext(name) == ".js" {
			for _, pattern := range imports {
				for _, match := range pattern.FindAllSubmatch(content, -1) {
					specifier := string(match[1])
					if !strings.HasPrefix(specifier, ".") {
						t.Errorf("non-relative Carbon import %s in %s", specifier, name)
						continue
					}
					if _, err := Assets.ReadFile(path.Join(path.Dir(name), specifier)); err != nil {
						t.Errorf("missing Carbon import %s from %s: %v", specifier, name, err)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCarbonChartsArePinnedEmbeddedAndSelfContained(t *testing.T) {
	content, err := Assets.ReadFile("vendor/carbon-charts/1.27.27/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Files   map[string]struct {
			Source string `json:"source"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "1.27.27" || len(manifest.Files) != 3 {
		t.Fatal("unexpected Carbon Charts distribution")
	}
	for name, record := range manifest.Files {
		content, err := Assets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(content)
		if !strings.HasPrefix(record.Source, "https://cdn.jsdelivr.net/npm/@carbon/charts@1.27.27/") || hex.EncodeToString(hash[:]) != record.SHA256 {
			t.Fatalf("Carbon Charts provenance or hash differs for %s", name)
		}
		if path.Ext(name) == ".css" && regexp.MustCompile(`(?i)url\(\s*["']?(?:https?:)?//`).Match(content) {
			t.Errorf("Carbon Charts CSS requires a remote resource: %s", name)
		}
	}
	bundle, err := Assets.ReadFile("vendor/carbon-charts/1.27.27/bundle.umd.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle[:1024]), `define(["exports"],`) || !strings.Contains(string(bundle[:1024]), `ze(J.Charts={})`) {
		t.Fatal("Carbon Charts UMD requires an unexpected external dependency")
	}
}

func TestCarbonGridMatchesProductiveTemplateClasses(test *testing.T) {
	grid, err := Assets.ReadFile("vendor/carbon/v2.67.0/grid.css")
	if err != nil {
		test.Fatal(err)
	}
	for _, class := range []string{".cds--grid", ".cds--row", ".cds--col-sm-4", ".cds--col-md-8", ".cds--col-lg-4", ".cds--col-lg-10"} {
		if !strings.Contains(string(grid), class) {
			test.Errorf("vendored grid lacks productive class %s", class)
		}
	}
	if strings.Contains(string(grid), ".bx--") {
		test.Fatal("legacy grid classes do not match Carbon v11 templates")
	}
}

func TestTemplatesReferenceEmbeddedAssetsAndSafeMarkup(t *testing.T) {
	references := regexp.MustCompile(`/assets/([^"'\s<>]+)`)
	unsafeExpression := regexp.MustCompile(`(?:x-[\w:-]+|@[^=\s]+|:[^=\s]+)="[^"]*\{\{`)
	inlineScript := regexp.MustCompile(`<script(?:\s[^>]*)?>`)
	templates, err := fs.Glob(Assets, "*.template")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range templates {
		content, err := Assets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range references.FindAllSubmatch(content, -1) {
			filename := strings.SplitN(strings.SplitN(string(match[1]), "?", 2)[0], "#", 2)[0]
			if _, err := Assets.ReadFile(filename); err != nil {
				t.Errorf("template %s references missing embedded file %s", name, filename)
			}
		}
		if unsafeExpression.Match(content) || strings.Contains(string(content), "<style") {
			t.Errorf("template %s contains interpolated expressions or inline styles", name)
		}
		for _, script := range inlineScript.FindAll(content, -1) {
			if !strings.Contains(string(script), " src=") {
				t.Errorf("template %s contains executable inline script", name)
			}
		}
	}
	if _, err := Assets.ReadFile("UnblockNeteaseMusic.template"); err == nil {
		t.Fatal("disabled legacy page is still embedded")
	}
}

func TestFirstPartyAssetsHaveNoRemoteRuntimeOrUnsafeUI(t *testing.T) {
	blocked := []string{"cdn." + "jsdelivr.net", "fonts." + "googleapis.com", "fonts." + "gstatic.com", "ui-" + "avatars.com", "unpkg" + ".com", "s81c" + ".com", "tail" + "wind", "daisy" + "ui", "remix" + "icon", "paci" + "fico"}
	thirdParty := map[string]bool{"js/alpine.min.js": true, "js/ace.js": true, "js/mode-toml.js": true, "js/xterm.js": true, "js/xterm-addon-attach.js": true, "js/xterm-addon-fit.js": true, "js/xterm-addon-web-links.js": true, "css/xterm.css": true}
	unsafeUI := regexp.MustCompile(`x-html|\b(?:alert|confirm|prompt)\s*\(|location\.reload\s*\(|window\.[\w$]+\s*=`)
	err := fs.WalkDir(Assets, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if name == "vendor" || name == "icons" {
				return fs.SkipDir
			}
			return nil
		}
		if thirdParty[name] || !slices.Contains([]string{".js", ".css", ".template", ".webmanifest", ".svg"}, path.Ext(name)) {
			return nil
		}
		content, err := Assets.ReadFile(name)
		if err != nil {
			return err
		}
		text := strings.ToLower(string(content))
		for _, fragment := range blocked {
			if strings.Contains(text, fragment) {
				t.Errorf("first-party runtime asset %s contains forbidden reference %s", name, fragment)
			}
		}
		if unsafeUI.Match(content) {
			t.Errorf("first-party runtime asset %s contains unsafe legacy UI", name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
