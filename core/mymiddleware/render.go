package mymiddleware

import (
	"github.com/labstack/echo/v5"
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

func (t *TemplateRender) Render(c *echo.Context, w io.Writer, name string, data interface{}) error {
	// Context 在这里虽然传入了，但在单纯的各种模板渲染中可能暂时用不到，除非你需要从 c 中获取额外信息注入模板
	return t.ExecuteTemplate(w, name, data)
}
