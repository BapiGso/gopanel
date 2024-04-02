package website

import (
	"bytes"
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
		return c.JSON(200, caddyConfig.Get("apps.http.servers."+c.QueryParam("name")))
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
			if err := caddyConfig.ReadConfig(io.Reader(bytes.NewBuffer(data))); err != nil {
				return err
			}
		} else {
			caddyConfig.Set("apps.http.servers."+c.QueryParam("name"), result)
		}
		if err := caddyConfig.WriteConfig(); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "OPTIONS":
		if err := caddyStop(); err != nil {
			return err
		}
		if c.QueryParam("status") == "restart" {
			if err := caddyStart(convertJSON(caddyConfig.Get("logging.logs.access_log.writer")), convertJSON(caddyConfig.Get("apps.http"))); err != nil {
				return err
			}
		}
		return c.JSON(200, "success")
	case "GET":
		return c.Render(http.StatusOK, "website.template", nil)
	}
	return echo.ErrMethodNotAllowed
}
