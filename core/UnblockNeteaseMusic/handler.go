package UnblockNeteaseMusic

import (
	"flag"
	"github.com/labstack/echo/v4"
	"strings"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "GET":
		return c.Render(200, "UnblockNeteaseMusic.template", nil)
	case "POST":
		if err := createCertificate(); err != nil {
			return err
		}
		if err := flag.CommandLine.Parse(strings.Fields(c.QueryParam("params"))); err != nil {
			return err
		}
		start()
		return c.JSON(200, "success")
	}
	return echo.ErrMethodNotAllowed
}
