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
	return ctx.Render(http.StatusOK, "security.template", map[string]any{
		"panelConfig":      state.Config.Panel,
		"settingsRevision": state.Revision,
	})
}
