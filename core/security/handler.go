package security

import (
	"github.com/labstack/echo/v5"
	"gopanel/core/config"
	"net/http"
)

func Index(c *echo.Context) error {

	switch c.Request().Method {
	case "GET":
		return c.Render(http.StatusOK, "security.template", map[string]any{
			"getPanelConfig": func(s string) any {
				return config.Get(s)
			},
		})
	case "POST":
		req := &struct {
			Port     string `form:"port"`
			Path     string `form:"path"`
			Username string `form:"username"`
			Password string `form:"password"`
		}{}
		if err := c.Bind(req); err != nil {
			return err
		}
		if err := config.Update(func(cfg *config.Config) {
			cfg.Panel.Port = req.Port
			cfg.Panel.Path = req.Path
			cfg.Panel.Username = req.Username
			cfg.Panel.Password = req.Password
		}); err != nil {
			return err
		}
		if err := restart(); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "PUT":
		if c.QueryParam("action") == "update" {
			if err := updateBinaryIfNeeded(); err != nil {
				return err
			}
		}
		if c.QueryParam("action") == "restart" {
			if err := restart(); err != nil {
				return err
			}
		}
		return c.JSON(200, "success")
	}
	return echo.ErrMethodNotAllowed
}
