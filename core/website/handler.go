package website

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"os"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "POST":
		if c.QueryParam("name") == "AllSetting" {
			return c.JSON(200, convertJSON(caddyConfig().AllSettings()))
		}
		return c.JSON(200, convertJSON(caddyConfig().Get("apps.http.servers."+c.QueryParam("name"))))
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
		//if c.QueryParam("name") == "AllSetting" {
		//	caddyConfig().SetConfigFile(string(data))
		//}
		//todo
		caddyConfig().Set("apps.http.servers."+c.QueryParam("name"), true)
		caddyConfig().Set(c.QueryParam("name"), result)
		if err := caddyConfig().WriteConfig(); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "OPTIONS":
		if err := caddy.Stop(); err != nil {
			return err
		}
		if c.QueryParam("status") == "restart" {
			conf, err := os.ReadFile("caddyConfig.json")
			if err != nil {
				return err
			}
			if err := caddy.Load(conf, true); err != nil {
				return err
			}
		}
		return c.JSON(200, "success")

	case "GET":
		return c.Render(http.StatusOK, "website.template", map[string]any{
			"websiteList": caddyConfig().Get("apps.http.servers"),
		})
	}
	return echo.ErrMethodNotAllowed
}

func convertJSON(data any) string {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(jsonData)
}
