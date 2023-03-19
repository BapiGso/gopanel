package core

import (
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gopanel/core/cron"
	"gopanel/core/database"
	"gopanel/core/docker"
	"gopanel/core/file"
	_ "gopanel/core/file"
	"gopanel/core/ftp"
	"gopanel/core/monitor"
	"gopanel/core/webssh"
	"io"
	"net/http"
	"text/template"
)

func (t *TemplateRender) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.Template.ExecuteTemplate(w, name, data)
}

func (c *Core) Route() {
	c.E.HideBanner = true
	c.E.Renderer = &TemplateRender{
		Template: template.Must(template.ParseFS(c.AssetsFS, "*.template")),
	}
	c.E.Logger.SetLevel(log.ERROR)

	c.E.Use(middleware.Logger())
	c.E.Use(middleware.Recover())
	c.E.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 3}))
	c.E.StaticFS("/assets", c.AssetsFS)
	c.E.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	c.E.GET("/", warning)
	c.E.GET("/:anywhere", warning)
	//c.E.GET(QueryPath(c.Db), loginGet)
	//c.E.POST(QueryPath(c.Db), loginPost)
	c.E.GET("/login", loginGet)
	c.E.POST("login", loginPost)
	g := c.E.Group("/admin")
	g.Use(isLogin)
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

}

func (c *Core) Run() {
	//log.Print(banner)
	c.E.Logger.Fatal(c.E.Start(":8848"))
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