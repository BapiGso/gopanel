package file

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type fileReq struct {
	Path string `query:"path" form:"path" json:"path"`
}

func FileGet(c echo.Context) error {
	F.read()
	return c.Render(http.StatusOK, "file.template", F)
}

func FilePost(c echo.Context) error {
	req := new(fileReq)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	F.path = req.Path
	return c.JSON(http.StatusOK, F)
}
