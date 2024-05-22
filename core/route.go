package core

import (
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"log/slog"
	"net/http"
	"os"
	"panel/core/UnblockNeteaseMusic"
	"panel/core/cron"
	"panel/core/docker"
	"panel/core/file"
	"panel/core/frps"
	"panel/core/login"
	"panel/core/monitor"
	"panel/core/security"
	"panel/core/term"
	"panel/core/unit"
	"panel/core/webdav"
	"panel/core/website"
	"strconv"
	"text/template"
)

func (c *Core) Route() {
	c.e.Validator = &unit.Validator{}
	c.e.Renderer = &unit.TemplateRender{
		Template: template.Must(template.ParseFS(c.assetsFS, "*.template")),
	}
	c.e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				slog.LogAttrs(context.Background(), slog.LevelError, v.Error.Error(),
					slog.String("Method", v.Method),
					slog.String("Url", v.URI),
				)
			}
			return nil
		},
	}))
	c.e.Use(middleware.Recover())
	c.e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 3}))
	c.e.HTTPErrorHandler = func(err error, c echo.Context) {
		c.JSON(400, err.Error())
	}

	c.e.Any(viper.GetString("panel.path"), login.Login, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(3))) //限制频率
	webdavMethods := []string{"GET", "HEAD", "POST", "OPTIONS", "PUT", "MKCOL", "DELETE", "PROPFIND", "PROPPATCH", "COPY", "MOVE", "REPORT", "LOCK", "UNLOCK"}
	c.e.Match(webdavMethods, "/webdav", webdav.FileSystem())
	c.e.Match(webdavMethods, "/webdav/*", webdav.FileSystem())
	// 静态资源
	c.e.StaticFS("/assets", c.assetsFS)
	//用于PWA的路径重写
	c.e.Pre(middleware.Rewrite(map[string]string{
		"/manifest.webmanifest": "/assets/manifest.webmanifest",
		"/sw.js":                "/assets/js/sw.js",
	}))
	// 后台路由
	admin := c.e.Group("/admin")
	admin.Use(echojwt.WithConfig(echojwt.Config{
		ErrorHandler: func(c echo.Context, err error) error {
			return c.Render(http.StatusTeapot, "warning.template", err)
		},
		SigningKey:  []byte(strconv.Itoa(os.Getpid())),
		TokenLookup: "cookie:panel_token",
		Skipper:     func(c echo.Context) bool { return login.Debug },
	}))
	admin.GET("/monitor", monitor.Index)
	admin.GET("/monitor/Stream", monitor.StreamInfo)
	admin.Any("/website", website.Index)
	admin.GET("/file", file.Index)
	admin.Any("/file/process", file.Process)
	admin.Any("/webdav", webdav.Index)
	admin.GET("/term", term.Index)
	admin.POST("/term", term.CreateTermHandler)
	admin.GET("/term/:id/data", term.LinkTermDataHandler)
	admin.GET("/term/resize", term.SetTermWindowSizeHandler)
	admin.GET("/security", security.Index)
	admin.GET("/cron", cron.Index)
	admin.GET("/docker", docker.Index)
	admin.Any("/frps", frps.Index)
	admin.Any("/UnblockNeteaseMusic", UnblockNeteaseMusic.Index)
	c.e.Start(viper.GetString("panel.port"))
}

func (c *Core) Close() {
	c.e.Close()
}
