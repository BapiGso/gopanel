package mymiddleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/httpapi"
	"gopanel/core/passwords"
)

func APIAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx *echo.Context) error {
		if debugBypassEnabled() || SessionManager.GetBool(ctx.Request().Context(), sessionAuthKey) {
			return next(ctx)
		}
		username, password, ok := ctx.Request().BasicAuth()
		settings := config.Snapshot().Panel
		validUsername := subtle.ConstantTimeCompare([]byte(username), []byte(settings.Username))
		if ok && settings.Username != "" && settings.Password != "" &&
			validUsername == 1 && passwords.VerifyPassword(settings.Password, password) {
			return next(ctx)
		}
		ctx.Response().Header().Set("WWW-Authenticate", `Basic realm="GoPanel API", charset="UTF-8"`)
		return httpapi.NewError(http.StatusUnauthorized, "unauthorized", "Use the panel session cookie or HTTP Basic credentials over HTTPS")
	}
}
