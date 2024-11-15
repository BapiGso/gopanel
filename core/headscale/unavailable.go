//go:build !linux && !darwin && !freebsd

package headscale

import "github.com/labstack/echo/v4"

func Index(c echo.Context) error {
	return c.JSON(200, map[string]string{"message": "Your operating system does not support"})
}
