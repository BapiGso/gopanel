//go:build !linux

package firewall

import "github.com/labstack/echo/v4"

func Index(c echo.Context) error {
	return c.Render(200, "unavailable.template", nil)
}
