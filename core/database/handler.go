package database

import (
	"github.com/labstack/echo/v5"
	"net/http"
)

func Index(c *echo.Context) error {
	return c.Render(http.StatusOK, "monitor.template", nil)
}
