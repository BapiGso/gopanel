package mymiddleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/httpapi"
)

func APIAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx *echo.Context) error {
		if debugBypassEnabled() || SessionManager.GetBool(ctx.Request().Context(), sessionAuthKey) {
			return next(ctx)
		}
		username, password, ok := ctx.Request().BasicAuth()
		settings := config.Snapshot().Panel
		usernameHash := sha256.Sum256([]byte(username))
		passwordHash := sha256.Sum256([]byte(password))
		expectedUsername := sha256.Sum256([]byte(settings.Username))
		expectedPassword := sha256.Sum256([]byte(settings.Password))
		valid := subtle.ConstantTimeCompare(usernameHash[:], expectedUsername[:]) & subtle.ConstantTimeCompare(passwordHash[:], expectedPassword[:])
		if ok && settings.Username != "" && settings.Password != "" && valid == 1 {
			return next(ctx)
		}
		ctx.Response().Header().Set("WWW-Authenticate", `Basic realm="GoPanel API", charset="UTF-8"`)
		return httpapi.NewError(http.StatusUnauthorized, "unauthorized", "Use the panel session cookie or HTTP Basic credentials over HTTPS")
	}
}
