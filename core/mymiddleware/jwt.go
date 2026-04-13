package mymiddleware

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v5"
)

const (
	sessionCookieName = "panel_token"
	sessionAuthKey    = "authenticated"
)

var SessionManager = func() *scs.SessionManager {
	manager := scs.New()
	manager.Lifetime = 7 * 24 * time.Hour
	manager.Cookie.Name = sessionCookieName
	manager.Cookie.HttpOnly = true
	manager.Cookie.Path = "/"
	manager.Cookie.SameSite = http.SameSiteLaxMode
	manager.Cookie.Secure = true
	return manager
}()

var Session = echo.WrapMiddleware(SessionManager.LoadAndSave)

var JWT echo.MiddlewareFunc = func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if debugBypassEnabled() {
			return next(c)
		}
		if SessionManager.GetBool(c.Request().Context(), sessionAuthKey) {
			return next(c)
		}
		return c.Render(http.StatusUnauthorized, "warning.template", map[string]string{
			"message": "unauthorized",
			"ip":      c.RealIP(),
		})
	}
}
