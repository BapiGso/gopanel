package mymiddleware

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/labstack/echo/v5"

	"gopanel/assets"
)

var DefaultTemplateRender = &TemplateRender{
	Template: template.Must(template.New("").Funcs(template.FuncMap{"icon": icon}).ParseFS(assets.Assets, "*.template")),
}

var iconRoot = regexp.MustCompile(`<svg\b[^>]*>`)
var iconAttributes = regexp.MustCompile(`\s+(?:width|height|class|aria-hidden|focusable)="[^"]*"`)
var iconName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func icon(name string, size int) (template.HTML, error) {
	if path.Base(name) != name || !iconName.MatchString(name) || size < 8 || size > 64 {
		return "", fmt.Errorf("invalid icon name or size")
	}
	directory := fmt.Sprintf("icons/carbon/%d/", size)
	if size == 64 {
		directory = "icons/carbon/pictograms/"
	}
	content, err := assets.Assets.ReadFile(directory + name + ".svg")
	if err != nil {
		return "", err
	}
	svg := string(content)
	opening := iconRoot.FindString(svg)
	if opening == "" {
		return "", fmt.Errorf("icon has no SVG root")
	}
	attributes := iconAttributes.ReplaceAllString(opening, "")
	attributes = strings.Replace(attributes, "<svg", fmt.Sprintf(`<svg class="ui-icon" width="%d" height="%d" aria-hidden="true" focusable="false"`, size, size), 1)
	svg = strings.Replace(svg, opening, attributes, 1)
	return template.HTML(svg), nil
}

type TemplateRender struct {
	*template.Template //渲染模板
}

func (t *TemplateRender) Render(c *echo.Context, w io.Writer, name string, data any) error {
	if data == nil {
		data = map[string]any{}
	}
	page, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal page %s: %w", name, err)
	}
	title := "GoPanel"
	public := false
	if values, ok := data.(map[string]any); ok {
		if value, ok := values["title"].(string); ok && value != "" {
			title = value
		}
		public, _ = values["public"].(bool)
	}
	language := "en"
	if name == "login.template" || name == "monitor.template" {
		language = "zh-CN"
	}
	return t.ExecuteTemplate(w, name, struct {
		Data     any
		Page     string
		Title    string
		Language string
		Public   bool
		Monitor  bool
	}{Data: data, Page: string(page), Title: title, Language: language, Public: public, Monitor: name == "monitor.template"})
}
