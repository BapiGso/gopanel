package cron

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func Index(c echo.Context) error {
	return c.Render(http.StatusOK, "cron.template", nil)
}
