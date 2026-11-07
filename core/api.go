package core

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"gopanel/core/cron"
	"gopanel/core/docker"
	"gopanel/core/file"
	"gopanel/core/firewall"
	"gopanel/core/frp"
	"gopanel/core/headscale"
	"gopanel/core/httpapi"
	"gopanel/core/mymiddleware"
	"gopanel/core/security"
	"gopanel/core/term"
	"gopanel/core/webdav"
	"gopanel/core/website"
)

func (panel *Core) setupAPIRoutes() {
	group := panel.e.Group(httpapi.Prefix)
	group.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)), mymiddleware.APIAuth, httpapi.Policy)
	router := httpapi.NewRouter(group)
	website.RegisterAPI(router)
	frp.RegisterAPI(router)
	webdav.RegisterAPI(router)
	security.RegisterAPI(router)
	cron.RegisterAPI(router)
	file.DefaultHandler().RegisterAPI(router)
	docker.RegisterAPI(router)
	firewall.RegisterAPI(router)
	term.RegisterAPI(router)
	panel.monitoring.RegisterAPI(router, panel.e)
	headscale.RegisterAPI(router)
	index := func(ctx *echo.Context) error {
		return httpapi.OK(ctx, map[string]any{"version": "v1", "openapi": httpapi.Prefix + "/openapi.json"})
	}
	router.Handle(http.MethodGet, "/", index, httpapi.Operation{ID: "getAPI", Summary: "Discover the API version and OpenAPI document", Response: map[string]string{}})
	group.GET("", index)
	group.GET("/openapi.json", func(ctx *echo.Context) error {
		return ctx.JSON(http.StatusOK, router.Document())
	})
}
