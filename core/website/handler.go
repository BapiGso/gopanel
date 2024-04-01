package website

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "POST":
		if c.QueryParam("name") == "AllSetting" {
			return c.JSON(200, caddyConfig.AllSettings())
		}
		return c.JSON(200, caddyConfig.Get("servers."+c.QueryParam("name")))
	case "PUT":
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}
		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}
		if c.QueryParam("name") == "AllSetting" {
			caddyConfig.SetConfigFile(string(data))
		}
		caddyConfig.Set("servers."+c.QueryParam("name"), result)
		if err := caddyConfig.WriteConfig(); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "OPTIONS":
		if err := caddyStop(); err != nil {
			return err
		}
		if c.QueryParam("status") == "restart" {
			if err := caddyStart(convertJSON(caddyConfig.AllSettings())); err != nil {
				return err
			}
		}
		return c.JSON(200, "success")
	case "GET":
		return c.Render(http.StatusOK, "website.template", map[string]any{
			"websiteList": caddyConfig.Get("servers"),
		})
	}
	return echo.ErrMethodNotAllowed
}
