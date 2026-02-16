package core

import (
	"context"
	"gopanel/core/cron"
	"gopanel/core/docker"
	"gopanel/core/file"
	"gopanel/core/firewall"
	"gopanel/core/frp"
	"gopanel/core/headscale"
	"gopanel/core/login"
	"gopanel/core/monitor"
	"gopanel/core/mymiddleware"
	"gopanel/core/security"
	"gopanel/core/term"
	"gopanel/core/webdav"
	"gopanel/core/website"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/spf13/viper"
)

func (c *Core) Route() {
	c.e.Validator = &mymiddleware.Validator{}
	c.e.Renderer = mymiddleware.DefaultTemplateRender

	c.e.Use(middleware.RequestLogger())
	c.e.Use(middleware.Recover())
	c.e.Use(middleware.Gzip())

	c.e.HTTPErrorHandler = func(c *echo.Context, err error) {

		c.JSON(400, err.Error())
	}
	//限制频率
	c.e.Any(viper.GetString("panel.path"), login.Login, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(3)))

	c.e.Match([]string{"GET", "HEAD", "POST", "OPTIONS", "PUT", "MKCOL",
		"DELETE", "PROPFIND", "PROPPATCH", "COPY", "MOVE", "REPORT",
		"LOCK", "UNLOCK"}, "/webdav*", webdav.WebDav)

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
	admin.GET("/monitor", monitor.Index)
	admin.Any("/website", website.Index)
	admin.GET("/file", file.Index)
	admin.Any("/file/process", file.Process)
	admin.Any("/webdav", webdav.Index)
	admin.GET("/term", term.Index)
	admin.POST("/term", term.CreateTermHandler)
	admin.GET("/term/:id/data", term.LinkTermDataHandler)
	admin.GET("/term/resize", term.SetTermWindowSizeHandler)
	admin.Any("/security", security.Index)
	admin.Any("/cron", cron.Index)
	admin.Any("/docker", docker.Index)
	admin.Any("/frp", frp.Index)
	// Headscale RESTful 路由
	admin.Any("/headscale", headscale.Index) // 获取页面
	admin.Any("/firewall", firewall.Index)
	//admin.Any("/UnblockNeteaseMusic", UnblockNeteaseMusic.Index)
	//c.e.Start(viper.GetString("panel.port"))
	//c.e.StartTLS(viper.GetString("panel.port"), []byte(certPEM), []byte(keyPEM))
	sc := echo.StartConfig{Address: viper.GetString("panel.port")}
	if err := sc.StartTLS(context.Background(), c.e, certPEM, keyPEM); err != nil {
		c.e.Logger.Error("failed to start server", "error", err)
	}
}
