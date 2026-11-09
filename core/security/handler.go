package security

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
)

func Index(ctx *echo.Context) error {
	state, err := config.Read()
	if err != nil {
		return err
	}
	version := "Development build"
	if current, known := currentVersion(); known {
		version = current.String()
	}
	return ctx.Render(http.StatusOK, "security.template", map[string]any{
		"panelConfig": map[string]any{
			"Username": state.Config.Panel.Username,
			"Port":     state.Config.Panel.Port,
			"Path":     state.Config.Panel.Path,
		},
		"version":          version,
		"settingsRevision": state.Revision,
	})
}
