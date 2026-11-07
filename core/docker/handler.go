package docker

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func Index(ctx *echo.Context) error {
	return ctx.Render(http.StatusOK, "docker.template", nil)
}
