//go:build linux
// +build linux

package firewall

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "GET":
		return c.Render(http.StatusOK, "firewall.template", nil)
	}
	return echo.ErrMethodNotAllowed
}
