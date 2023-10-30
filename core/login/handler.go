package login

import "github.com/labstack/echo/v4"

func Login(c echo.Context) error {
	req := &struct {
		Path     string
		User     string
		PassWord string
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	switch c.Request().Method {
	case "GET":
		if req.Path != c.Get("conf").(string) {
			return c.Render(400, "warning.template", nil)
		}
		return c.Render(200, "login.template", nil)
	case "POST":

	}
	return c.NoContent(400)
}
