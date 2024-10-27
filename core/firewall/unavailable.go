// core/firewall/unavailable.go
//go:build !linux
// +build !linux

package firewall

import "github.com/labstack/echo/v4"

func Index(c echo.Context) error {
	return c.JSON(200, map[string]string{"message": "Your operating system does not support"})
}
