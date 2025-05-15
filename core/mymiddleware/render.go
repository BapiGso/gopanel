package mymiddleware

import (
	"github.com/labstack/echo/v4"
	"gopanel/assets"
	"io"
	"text/template"
)

var DefaultTemplateRender = &TemplateRender{
	Template: template.Must(template.ParseFS(assets.Assets, "*.template")),
}

type TemplateRender struct {
	*template.Template //渲染模板
}

func (t *TemplateRender) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.ExecuteTemplate(w, name, data)
}
