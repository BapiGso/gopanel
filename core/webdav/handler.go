package webdav

import (
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
	state, err := config.Read()
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "webdav.template", map[string]any{
		"webdavConfig":     state.Config.WebDAV,
		"settingsRevision": state.Revision,
	})
}

func FileSystem() echo.HandlerFunc {
	return echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "Not authorized", 401)
			return
		}
		settings := config.Snapshot().WebDAV
		if username != settings.Username || password != settings.Password {
			http.Error(w, "Not authorized", 401)
			return
		}
		if !settings.Enable {
			http.Error(w, "Not enable", http.StatusForbidden)
			return
		}
		srv.ServeHTTP(w, r)
	}))
}
