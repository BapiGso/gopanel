package webdav

import (
	"fmt"
	"github.com/labstack/echo/v5"
	"golang.org/x/net/webdav"
	"gopanel/core/config"
	"net/http"
)

var srv = &webdav.Handler{
	Prefix:     "/webdav",
	FileSystem: webdav.Dir("/"),
	LockSystem: webdav.NewMemLS(),
}

func Index(c *echo.Context) error {
	switch c.Request().Method {
	case "GET":
		return c.Render(http.StatusOK, "webdav.template", map[string]any{
			"getPanelConfig": func(s string) any {
				return config.Get(s)
			},
		})
	case "POST":
		req := &struct {
			Enable   string `form:"enable"`
			Username string `form:"username"`
			Password string `form:"password"`
		}{
		}
		if err := c.Bind(req); err != nil {
			return err
		}
		if err := config.Update(func(cfg *config.Config) {
			cfg.WebDAV.Enable = req.Enable == "on"
			cfg.WebDAV.Username = req.Username
			cfg.WebDAV.Password = req.Password
		}); err != nil {
			return err
		}
		return c.JSON(204, "success")
	}
	return echo.ErrMethodNotAllowed
}

func FileSystem() echo.HandlerFunc {
	return echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(config.String("webdav.username"))
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "Not authorized", 401)
			return
		}
		if username != config.String("webdav.username") || password != config.String("webdav.password") {
			http.Error(w, "Not authorized", 401)
			return
		}
		if !config.Bool("webdav.enable") {
			http.Error(w, "Not enable", 401)
			return
		}
		srv.ServeHTTP(w, r)
	}))
}

func WebDav(c *echo.Context) error {
	fmt.Println(123)
	srv.ServeHTTP(c.Response(), c.Request())
	return nil
}
