package website

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "POST":
		if c.QueryParam("status") == "restart" {
			if err := start(); err != nil {
				return err
			}
		}
		if c.QueryParam("status") == "stop" {
			if err := caddy.Stop(); err != nil {
				return err
			}
		}
		if c.QueryParam("status") == "enable" {
			viper.Set("enable.caddy", !viper.Get("enable.caddy").(bool))
		}
		return c.JSON(200, "success")
	case "PUT":
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile("Caddyfile", data, 0644); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "GET":
		file, err := os.ReadFile("Caddyfile")
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "website.template", map[string]any{
			"caddyFile":   string(file),
			"caddyEnable": viper.Get("enable.caddy").(bool),
		})
	}
	return echo.ErrMethodNotAllowed
}
