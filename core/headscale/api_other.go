//go:build !linux && !darwin && !freebsd

package headscale

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

func getStatus(ctx *echo.Context) error {
	return httpapi.NewError(http.StatusNotImplemented, "not_supported", "Embedded Headscale is supported only on Linux, macOS and FreeBSD")
}

func start(ctx *echo.Context) error {
	return getStatus(ctx)
}

func stop(ctx *echo.Context) error {
	return getStatus(ctx)
}
