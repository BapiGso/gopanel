package login

import (
	"github.com/labstack/echo/v5"
	"gopanel/core/config"
	"gopanel/core/mymiddleware"
)

func Login(c *echo.Context) error {
	switch c.Request().Method {
	case "GET":
		return c.Render(200, "login.template", nil)
	case "POST":
		req := &struct {
			Username string `form:"username" validate:"required,min=1,max=200"`
			Password string `form:"password" validate:"required,min=8,max=200"`
		}{}
		if err := c.Bind(req); err != nil {
			return err
		}
		if err := c.Validate(req); err != nil {
			return err
		}
		if req.Username == config.String("panel.username") && req.Password == config.String("panel.password") {
			if err := mymiddleware.SessionManager.RenewToken(c.Request().Context()); err != nil {
				return err
			}
			mymiddleware.SessionManager.Put(c.Request().Context(), "authenticated", true)
			return c.Redirect(302, "/admin/monitor")
		}
		return echo.ErrUnauthorized
	}
	return echo.ErrMethodNotAllowed
}
