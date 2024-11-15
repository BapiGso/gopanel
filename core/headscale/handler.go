package headscale

import (
	"github.com/juanfont/headscale/cmd/headscale/cli"
	"github.com/labstack/echo/v4"
	"net/http"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "POST":
		cli.Execute()
	case "PUT":

	case "GET":
		return c.Render(http.StatusOK, "headscale.template", map[string]any{})
	}

	return echo.ErrMethodNotAllowed
}
