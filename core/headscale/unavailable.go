//go:build !linux && !darwin && !freebsd

package headscale

import "github.com/labstack/echo/v4"

func Index(c echo.Context) error {

	return c.Render(200, "unavailable.template", nil)
}
