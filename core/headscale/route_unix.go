//go:build linux || darwin || freebsd

package headscale

import "github.com/labstack/echo/v5"

func Handler() echo.HandlerFunc {
	return Index
}
