package unit

import (
	"github.com/labstack/echo/v4"
	"io"
	"text/template"
)

type TemplateRender struct {
	Template *template.Template //渲染模板
}

func (t *TemplateRender) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.Template.ExecuteTemplate(w, name, data)
}
