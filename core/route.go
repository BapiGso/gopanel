package core

import (
	"fmt"
	"github.com/labstack/echo-contrib/session"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"panel/core/cron"
	"panel/core/database"
	"panel/core/docker"
	"panel/core/file"
	"panel/core/ftp"
	"panel/core/monitor"
	"panel/core/store"
	"panel/core/unit"
	"panel/core/webssh"
	"text/template"
)

func (c *Core) Route() {
	c.e.Validator = &unit.Validator{}
	c.e.Renderer = &unit.TemplateRender{
		Template: template.Must(template.ParseFS(c.assetsFS, "*.template")),
	}

	c.e.Use(middleware.Logger())
	c.e.Use(middleware.Recover())
	c.e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 3}))
	//将配置文件暴露到content中
	c.e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ec echo.Context) error {
			ec.Set("conf", c.Conf)
			return next(ec)
		}
	})

	c.e.GET("/test", func(c echo.Context) error {
		return c.Render(200, "header.template", nil)
	})
	c.e.GET("/", warning)
	c.e.Any("/:anywhere", warning)
	g := c.e.Group("/admin")
	g.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  c.Conf.JWTKey,
		TokenLookup: "cookie:gopanel_token",
	}))
	g.GET("/home", home)
	g.GET("/site", site)
	g.GET("/database", database.Index)
	g.GET("/ftp", ftp.Index)
	g.GET("/file", file.FileGet)
	g.POST("/file", file.FilePost)
	g.GET("/term", webssh.Index)
	g.POST("/term", webssh.CreateTermHandler)
	g.GET("/term/:id/data", webssh.LinkTermDataHandler)
	g.POST("/term/:id/windowsize", webssh.SetTermWindowSizeHandler)
	g.GET("/monitor", monitor.Index)
	g.GET("/monitorStream", monitor.StreamInfo)
	g.GET("/docker", docker.Index)
	g.GET("/cron", cron.Index)
	g.GET("/store", store.Index)
	c.e.StaticFS("/assets", c.assetsFS)
}

func (c *Core) Run() {
	fmt.Printf(banner, "")
	c.e.Logger.Fatal(c.e.Start(":8848"))
}

func isLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//登录页面不用这个中间件
		//if c.Path() == loginpath || c.Path() == "/" {
		//	return next(c)
		//}
		//后台页面没有cookie的全部跳去登录
		sess, _ := session.Get("session", c)
		if sess.Values["isLogin"] != true {
			return c.Redirect(http.StatusFound, "/")
		}
		return next(c)
	}
}
