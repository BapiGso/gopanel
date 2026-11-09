package login

import (
	"crypto/sha256"
	"crypto/subtle"

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
		settings := config.Snapshot().Panel
		if credentialsMatch(req.Username, req.Password, settings.Username, settings.Password) {
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

func credentialsMatch(username, password, expectedUsername, expectedPassword string) bool {
	userHash := sha256.Sum256([]byte(username))
	expectedUserHash := sha256.Sum256([]byte(expectedUsername))
	passwordHash := sha256.Sum256([]byte(password))
	expectedPasswordHash := sha256.Sum256([]byte(expectedPassword))
	userMatch := subtle.ConstantTimeCompare(userHash[:], expectedUserHash[:])
	passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:])
	return userMatch&passwordMatch == 1
}
