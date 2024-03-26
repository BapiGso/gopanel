package webdav

import (
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"golang.org/x/net/webdav"
	"net/http"
)

var srv = &webdav.Handler{
	Prefix:     "/webdav",
	FileSystem: webdav.Dir("/"),
	LockSystem: webdav.NewMemLS(),
}

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "GET":
		return c.Render(http.StatusOK, "webdav.template", map[string]any{
			"getPanelConfig": func(s string) any {
				return viper.Get(s)
			},
		})
	case "POST":
		req := &struct {
			Enable   string `form:"enable"`
			Username string `form:"username"`
			Password string `form:"password"`
		}{}
		if err := c.Bind(req); err != nil {
			return err
		}
		viper.Set("webdav.enable", req.Enable == "on")
		viper.Set("webdav.username", req.Username)
		viper.Set("webdav.password", req.Password)
		if err := viper.WriteConfig(); err != nil {
			return err // 处理错误
		}
		return c.JSON(204, "success")
	}
	return echo.ErrMethodNotAllowed
}

func FileSystem() echo.HandlerFunc {
	return echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "Not authorized", 401)
			return
		}
		if username != viper.GetString("webdav.username") || password != viper.GetString("webdav.password") {
			http.Error(w, "Not authorized", 401)
			return
		}
		if !viper.GetBool("webdav.enable") {
			http.Error(w, "Not enable", 401)
			return
		}
		srv.ServeHTTP(w, r)
	}))
}
