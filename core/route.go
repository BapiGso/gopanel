package core

import (
	"context"
	"gopanel/core/config"
	"gopanel/core/cron"
	"gopanel/core/docker"
	"gopanel/core/file"
	"gopanel/core/firewall"
	"gopanel/core/frp"
	"gopanel/core/headscale"
	"gopanel/core/httpapi"
	"gopanel/core/login"
	"gopanel/core/mymiddleware"
	"gopanel/core/security"
	"gopanel/core/term"
	"gopanel/core/webdav"
	"gopanel/core/website"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func (c *Core) Route() {
	if c.startupError != nil {
		c.e.Logger.Error("failed to initialize panel", "error", c.startupError)
		return
	}
	c.setupRoutes()

	certPEM, keyPEM, err := loadOrCreateCertificate()
	if err != nil {
		c.e.Logger.Error("failed to load or create tls certificate", "error", err)
		return
	}

	settings := config.Snapshot().Panel
	printStartupInfo(settings)
	go c.autostartServices()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c.monitoring.Start(ctx)
	sc := echo.StartConfig{Address: settings.Port}
	if err := sc.StartTLS(ctx, c.e, certPEM, keyPEM); err != nil {
		c.e.Logger.Error("failed to start server", "error", err)
	}
}

func (c *Core) setupRoutes() {
	c.e.Validator = &mymiddleware.Validator{}
	c.e.Renderer = mymiddleware.DefaultTemplateRender

	c.e.Use(mymiddleware.Session)
	c.e.Use(middleware.RequestLogger())
	c.e.Use(middleware.Recover())
	c.e.Use(middleware.Gzip())

	c.e.HTTPErrorHandler = func(c *echo.Context, err error) {
		response, _ := echo.UnwrapResponse(c.Response())
		if err == nil || response != nil && response.Committed {
			return
		}
		if httpapi.IsRequest(c) {
			httpapi.ErrorHandler(c, err)
			return
		}
		if httpErr, ok := err.(*echo.HTTPError); ok {
			_ = c.JSON(httpErr.Code, httpErr.Message)
			return
		}
		if status := echo.StatusCode(err); status != 0 {
			_ = c.JSON(status, http.StatusText(status))
			return
		}
		_ = c.JSON(http.StatusInternalServerError, err.Error())
	}
	c.setupAPIRoutes()
	//限制频率
	c.e.Any(config.Snapshot().Panel.Path, login.Login, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(3)))

	c.e.Match([]string{"GET", "HEAD", "POST", "OPTIONS", "PUT", "MKCOL",
		"DELETE", "PROPFIND", "PROPPATCH", "COPY", "MOVE", "REPORT",
		"LOCK", "UNLOCK"}, "/webdav*", webdav.FileSystem())

	// 静态资源
	c.e.StaticFS("/assets", c.assetsFS)
	//c.e.Group("/assets", middleware.StaticWithConfig(middleware.StaticConfig{
	//	Skipper: func(c *echo.Context) bool {
	//		c.Response().Header().Set("Cache-Control", "public, max-age=86400")
	//		return false
	//	},
	//	Filesystem: c.assetsFS,
	//}))
	//用于PWA的路径重写
	c.e.Pre(middleware.Rewrite(map[string]string{
		"/manifest.webmanifest": "/assets/manifest.webmanifest",
		"/sw.js":                "/assets/js/sw.js",
	}))
	// 后台路由
	admin := c.e.Group("/admin")
	admin.Use(mymiddleware.JWT)
	admin.GET("/monitor", c.monitoring.Index)
	admin.GET("/website", website.Index)
	admin.GET("/file", file.Index)
	admin.GET("/webdav", webdav.Index)
	admin.GET("/term", term.Index)
	admin.GET("/security", security.Index)
	admin.GET("/cron", cron.Index)
	admin.GET("/docker", docker.Index)
	admin.GET("/frp", frp.Index)
	// Headscale RESTful 路由
	admin.GET("/headscale", headscale.Handler()) // 获取页面
	admin.GET("/firewall", firewall.Index)
	//admin.Any("/UnblockNeteaseMusic", UnblockNeteaseMusic.Index)
}
