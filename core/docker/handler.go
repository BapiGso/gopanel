package docker

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func Index(c echo.Context) error {
	return c.Render(http.StatusOK, "docker.template", nil)
}
