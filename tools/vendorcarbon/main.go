package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

type fileRecord struct {
	Source string `json:"source"`
	SHA256 string `json:"sha256"`
}

type manifest struct {
	Version    string                `json:"version"`
	SourceURL  string                `json:"source_url"`
	Components []string              `json:"components"`
	Icons      []string              `json:"icons"`
	Pictograms []string              `json:"pictograms,omitempty"`
	Charts     string                `json:"charts_version,omitempty"`
	Transforms map[string]string     `json:"transforms"`
	Files      map[string]fileRecord `json:"files"`
}

type download struct {
	Name     string
	Source   string
	Content  []byte
	Err      error
	Fallback string
}

type vendorOptions struct {
	Version, Components, Icons, HeaderIcons, IconVersion                      string
	Pictograms, PictogramVersion, ChartsVersion, StylesURL, Root, SourceCache string
}

type statusError struct {
	status int
	source string
}

func (err *statusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", err.status, err.source)
}

var importPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bfrom\s*["']([^"']+)["']`),
	regexp.MustCompile(`\bimport\s*(?:\(\s*)?["']([^"']+)["']`),
}

var sourceMap = regexp.MustCompile(`(?m)//[#@]\s*sourceMappingURL=[^\r\n]*|/\*[#@]\s*sourceMappingURL=.*?\*/`)
var fontFace = regexp.MustCompile(`(?s)@font-face\s*\{[^}]*\}`)
var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var versionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

func main() {
	var options vendorOptions
	flag.StringVar(&options.Version, "version", "v2.67.0", "Pinned Carbon Web Components version")
	flag.StringVar(&options.Components, "components", "", "Comma-separated component entry names")
	flag.StringVar(&options.Icons, "icons", "", "Comma-separated Carbon 16px icon names")
	flag.StringVar(&options.HeaderIcons, "header-icons", "", "Comma-separated Carbon 20px global-action icon names")
	flag.StringVar(&options.IconVersion, "icon-version", "11.92.0", "Pinned @carbon/icons version")
	flag.StringVar(&options.Pictograms, "pictograms", "", "Comma-separated empty-state pictogram names")
	flag.StringVar(&options.PictogramVersion, "pictogram-version", "12.88.0", "Pinned @carbon/pictograms version")
	flag.StringVar(&options.ChartsVersion, "charts-version", "", "Optional pinned @carbon/charts UMD distribution")
	flag.StringVar(&options.StylesURL, "styles-url", "https://1.www.s81c.com/common/carbon-for-ibm-dotcom/tag/v1/latest/", "Style-helper directory URL")
	flag.StringVar(&options.Root, "root", "assets", "Asset output directory")
	flag.StringVar(&options.SourceCache, "source-cache", "", "Optional offline directory of source files named by SHA-256 of their source URL")
	flag.Parse()
	if err := vendor(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func names(input string) ([]string, error) {
	result := []string{}
	for _, name := range strings.Split(input, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !identifier.MatchString(name) {
			return nil, fmt.Errorf("invalid component or icon name %q", name)
		}
		result = append(result, name)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

func imports(content []byte) []string {
	result := []string{}
	for _, pattern := range importPatterns {
		for _, match := range pattern.FindAllSubmatch(content, -1) {
			result = append(result, string(match[1]))
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func fetch(client *http.Client, files []download) {
	var workers sync.WaitGroup
	limit := make(chan struct{}, 8)
	for index := range files {
		workers.Add(1)
		go func() {
			defer workers.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			file := &files[index]
			for attempt := 0; attempt < 3; attempt++ {
				file.Content, file.Err = get(client, file.Source)
				var status *statusError
				if errors.As(file.Err, &status) && status.status == http.StatusNotFound && file.Fallback != "" {
					file.Source, file.Fallback = file.Fallback, ""
					file.Content, file.Err = get(client, file.Source)
				}
				if file.Err == nil {
					return
				}
				time.Sleep(time.Duration(attempt+1) * time.Second)
			}
		}()
	}
	workers.Wait()
}

func get(client *http.Client, source string) ([]byte, error) {
	response, err := client.Get(source)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, &statusError{status: response.StatusCode, source: source}
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if len(content) == 0 || len(content) >= 16<<20 {
		return nil, fmt.Errorf("empty or oversized asset: %s", source)
	}
	return content, nil
}

func vendor(options vendorOptions) error {
	version, componentList, iconList, iconVersion, stylesURL, root := options.Version, options.Components, options.Icons, options.IconVersion, options.StylesURL, options.Root
	if !versionPattern.MatchString(version) || !versionPattern.MatchString("v"+iconVersion) {
		return fmt.Errorf("versions must be exact semantic versions")
	}
	components, err := names(componentList)
	if err != nil {
		return err
	}
	if len(components) == 0 {
		return fmt.Errorf("provide -components")
	}
	icons, err := names(iconList)
	if err != nil {
		return err
	}
	headerIcons, err := names(options.HeaderIcons)
	if err != nil {
		return err
	}
	pictograms, err := names(options.Pictograms)
	if err != nil {
		return err
	}
	if len(pictograms) > 0 && !versionPattern.MatchString("v"+options.PictogramVersion) {
		return fmt.Errorf("pictogram version must be an exact semantic version")
	}
	if options.ChartsVersion != "" && !versionPattern.MatchString("v"+options.ChartsVersion) {
		return fmt.Errorf("charts version must be an exact semantic version")
	}
	base := "https://1.www.s81c.com/common/carbon/web-components/version/" + version + "/"
	baseURL, err := url.Parse(base)
	if err != nil {
		return err
	}
	prefix := "vendor/carbon/" + version + "/"
	client := &http.Client{Timeout: 60 * time.Second}
	if options.SourceCache != "" {
		client.Transport = sourceCacheTransport{directory: options.SourceCache}
	}
	files := map[string]download{}
	pending := []download{}
	for _, component := range components {
		pending = append(pending, download{Name: prefix + component + ".min.js", Source: base + component + ".min.js"})
	}
	for len(pending) > 0 {
		fetch(client, pending)
		next := map[string]download{}
		for _, file := range pending {
			if file.Err != nil {
				return fmt.Errorf("download %s: %w", file.Source, file.Err)
			}
			file.Content = sourceMap.ReplaceAll(file.Content, nil)
			files[file.Name] = file
		}
		for _, file := range pending {
			currentURL, err := url.Parse(file.Source)
			if err != nil {
				return err
			}
			for _, specifier := range imports(file.Content) {
				if !strings.HasPrefix(specifier, ".") {
					return fmt.Errorf("non-relative import %q in %s", specifier, file.Source)
				}
				reference, err := url.Parse(specifier)
				if err != nil {
					return err
				}
				resolved := currentURL.ResolveReference(reference)
				if resolved.Scheme != baseURL.Scheme || resolved.Host != baseURL.Host || !strings.HasPrefix(resolved.Path, baseURL.Path) || resolved.RawQuery != "" || resolved.Fragment != "" {
					return fmt.Errorf("import escapes pinned distribution: %s", resolved)
				}
				name := prefix + strings.TrimPrefix(resolved.Path, baseURL.Path)
				if _, exists := files[name]; !exists {
					next[name] = download{Name: name, Source: resolved.String()}
				}
			}
		}
		pending = pending[:0]
		for _, file := range next {
			pending = append(pending, file)
		}
		slices.SortFunc(pending, func(first, second download) int { return strings.Compare(first.Name, second.Name) })
	}
	additional := []download{}
	additional = append(additional, download{Name: prefix + "LICENSE", Source: "https://cdn.jsdelivr.net/npm/@carbon/web-components@" + strings.TrimPrefix(version, "v") + "/LICENSE"})
	for _, style := range []string{"themes.css", "grid.css", "plex.css"} {
		additional = append(additional, download{Name: prefix + style, Source: strings.TrimRight(stylesURL, "/") + "/" + style})
	}
	fontBase := "https://unpkg.com/@ibm/plex@6.4.1/"
	fontPaths := []string{
		"IBM-Plex-Sans/fonts/split/woff2/IBMPlexSans-Light-Latin1.woff2",
		"IBM-Plex-Sans/fonts/split/woff2/IBMPlexSans-Regular-Latin1.woff2",
		"IBM-Plex-Sans/fonts/split/woff2/IBMPlexSans-SemiBold-Latin1.woff2",
		"IBM-Plex-Mono/fonts/split/woff2/IBMPlexMono-Regular-Latin1.woff2",
	}
	for _, font := range fontPaths {
		additional = append(additional, download{Name: "vendor/carbon/fonts/" + path.Base(font), Source: fontBase + font})
	}
	additional = append(additional, download{Name: "vendor/carbon/fonts/OFL.txt", Source: fontBase + "LICENSE.txt"})
	iconBase := "https://cdn.jsdelivr.net/npm/@carbon/icons@" + iconVersion + "/"
	for _, icon := range icons {
		additional = append(additional, download{Name: "icons/carbon/16/" + icon + ".svg", Source: iconBase + "svg/16/" + icon + ".svg", Fallback: iconBase + "svg/32/" + icon + ".svg"})
	}
	for _, icon := range headerIcons {
		additional = append(additional, download{Name: "icons/carbon/20/" + icon + ".svg", Source: iconBase + "svg/20/" + icon + ".svg", Fallback: iconBase + "svg/32/" + icon + ".svg"})
	}
	additional = append(additional, download{Name: "icons/carbon/LICENSE", Source: iconBase + "LICENSE"})
	pictogramBase := "https://cdn.jsdelivr.net/npm/@carbon/pictograms@" + options.PictogramVersion + "/"
	for _, pictogram := range pictograms {
		additional = append(additional, download{Name: "icons/carbon/pictograms/" + pictogram + ".svg", Source: pictogramBase + "svg/" + pictogram + ".svg"})
	}
	if len(pictograms) > 0 {
		additional = append(additional, download{Name: "icons/carbon/pictograms/LICENSE", Source: pictogramBase + "LICENSE"})
	}
	chartPrefix := "vendor/carbon-charts/" + options.ChartsVersion + "/"
	chartBase := "https://cdn.jsdelivr.net/npm/@carbon/charts@" + options.ChartsVersion + "/"
	if options.ChartsVersion != "" {
		for name, source := range map[string]string{"bundle.umd.js": "dist/umd/bundle.umd.js", "styles.min.css": "dist/styles.min.css", "LICENSE": "LICENSE.md"} {
			additional = append(additional, download{Name: chartPrefix + name, Source: chartBase + source})
		}
	}
	fetch(client, additional)
	for _, file := range additional {
		if file.Err != nil {
			return fmt.Errorf("download %s: %w", file.Source, file.Err)
		}
		if file.Name == prefix+"plex.css" {
			file.Content = append(fontFace.ReplaceAll(file.Content, nil), []byte(localFonts)...)
		}
		if file.Name == prefix+"grid.css" {
			file.Content = []byte(strings.ReplaceAll(string(file.Content), "bx--", "cds--"))
		}
		file.Content = sourceMap.ReplaceAll(file.Content, nil)
		files[file.Name] = file
	}
	if options.ChartsVersion != "" {
		chartManifest := manifest{Version: options.ChartsVersion, SourceURL: chartBase, Files: map[string]fileRecord{}}
		var chartSize int
		for name, file := range files {
			if !strings.HasPrefix(name, chartPrefix) {
				continue
			}
			hash := sha256.Sum256(file.Content)
			chartManifest.Files[name] = fileRecord{Source: file.Source, SHA256: hex.EncodeToString(hash[:])}
			chartSize += len(file.Content)
		}
		content, err := json.MarshalIndent(chartManifest, "", "  ")
		if err != nil {
			return err
		}
		files[chartPrefix+"manifest.json"] = download{Name: chartPrefix + "manifest.json", Source: chartBase, Content: append(content, '\n')}
		fmt.Printf("Carbon Charts %s: %d files, %d bytes plus manifest (self-contained UMD)\n", options.ChartsVersion, len(chartManifest.Files), chartSize)
	}
	usedIcons := append(append(append([]string{}, icons...), headerIcons...), pictograms...)
	slices.Sort(usedIcons)
	record := manifest{
		Version: strings.TrimPrefix(version, "v"), SourceURL: base, Components: components, Icons: slices.Compact(usedIcons), Pictograms: pictograms, Charts: options.ChartsVersion,
		Transforms: map[string]string{
			"*.js,*.css": "Removed sourceMappingURL references because source maps are not shipped.",
			"grid.css":   "Renamed the legacy bx-- grid class prefix to cds-- for Carbon v11 template classes.",
			"plex.css":   "Replaced all upstream font-face rules with local Latin1 IBM Plex Sans 300/400/600 and IBM Plex Mono 400 (Plex 6.4.1), font-display: swap; retained upstream reset and typography.",
			"icons":      "Use size-specific 16px and 20px artwork when available; fall back to the 32px artwork only on HTTP 404. Empty-state pictograms use their original 64px artboards.",
		},
		Files: map[string]fileRecord{},
	}
	var size int
	for name, file := range files {
		if path.Clean(name) != name || strings.HasPrefix(name, "../") {
			return fmt.Errorf("unsafe output path %s", name)
		}
		filename := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filename, file.Content, 0644); err != nil {
			return err
		}
		hash := sha256.Sum256(file.Content)
		record.Files[name] = fileRecord{Source: file.Source, SHA256: hex.EncodeToString(hash[:])}
		size += len(file.Content)
	}
	content, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(prefix+"manifest.json")), content, 0644); err != nil {
		return err
	}
	fmt.Printf("Vendored Carbon %s: %d components, %d icon/pictogram names, %d files, %d bytes plus manifest\n", record.Version, len(components), len(record.Icons), len(files), size)
	return nil
}

const localFonts = `
@font-face{font-family:'IBM Plex Sans';font-style:normal;font-weight:300;font-display:swap;src:url('../fonts/IBMPlexSans-Light-Latin1.woff2') format('woff2')}
@font-face{font-family:'IBM Plex Sans';font-style:normal;font-weight:400;font-display:swap;src:url('../fonts/IBMPlexSans-Regular-Latin1.woff2') format('woff2')}
@font-face{font-family:'IBM Plex Sans';font-style:normal;font-weight:600;font-display:swap;src:url('../fonts/IBMPlexSans-SemiBold-Latin1.woff2') format('woff2')}
@font-face{font-family:'IBM Plex Mono';font-style:normal;font-weight:400;font-display:swap;src:url('../fonts/IBMPlexMono-Regular-Latin1.woff2') format('woff2')}
`
