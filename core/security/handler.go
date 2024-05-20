package security

import (
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "GET":
		return c.Render(http.StatusOK, "security.template", map[string]any{
			"getPanelConfig": func(s string) any {
				return viper.Get(s)
			},
		})
	case "POST":
		req := &struct {
			Port     string `form:"port"`
			Path     string `form:"path"`
			Username string `form:"username"`
			Password string `form:"password"`
		}{}
		if err := c.Bind(req); err != nil {
			return err
		}
		viper.Set("panel.port", req.Port)
		viper.Set("panel.path", req.Path)
		viper.Set("panel.username", req.Username)
		viper.Set("panel.password", req.Password)
		if err := viper.WriteConfig(); err != nil {
			return err // 处理错误
		}
		return c.JSON(200, "success")
	}
	return echo.ErrMethodNotAllowed
}
