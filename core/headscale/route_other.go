//go:build !linux && !darwin && !freebsd

package headscale

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func Handler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.Render(http.StatusOK, "unavailable.template", nil)
	}
}
