package monitor

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (service *Service) Index(ctx *echo.Context) error {
	return service.page(ctx, false)
}

func (service *Service) publicPage(ctx *echo.Context) error {
	if !service.settings.Public {
		return echo.NewHTTPError(http.StatusNotFound, "Public monitoring is disabled")
	}
	return service.page(ctx, true)
}

func (service *Service) page(ctx *echo.Context, public bool) error {
	base := "/api/v1/monitor"
	if public {
		base = "/api/v1/public/monitor"
	}
	return ctx.Render(http.StatusOK, "monitor.template", map[string]any{
		"title": service.settings.Title, "api_base": base, "public": public,
	})
}
