package website

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
)

func Index(ctx *echo.Context) error {
	settings, err := config.Read()
	if err != nil {
		return err
	}
	document, err := caddyFile.Read()
	if err != nil {
		return err
	}
	content, err := json.Marshal(document.Content)
	if err != nil {
		return err
	}
	return ctx.Render(http.StatusOK, "website.template", map[string]any{
		"caddyFile":        string(content),
		"caddyRevision":    document.Revision,
		"caddyEnable":      settings.Config.Enable.Caddy,
		"settingsRevision": settings.Revision,
	})
}
