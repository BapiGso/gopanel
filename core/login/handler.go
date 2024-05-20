package login

import (
	"flag"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"strconv"
	"time"
)

// debug use this function
var Debug = func() bool {
	debug := flag.Bool("debug", false, "enable debug mode")
	// 解析传入的命令行参数
	flag.Parse()
	return *debug
}()

func Login(c echo.Context) error {
	switch c.Request().Method {
	case "GET":
		user, ok := c.Get("user").(*jwt.Token)
		if ok && user.Valid {
			return c.Redirect(302, "/admin/monitor")
		}
		return c.Render(200, "login.template", nil)
	case "POST":
		req := &struct {
			Username string `form:"username" validate:"required,min=1,max=200"`
			Password string `form:"password" validate:"required,min=8,max=200"`
		}{}
		if err := c.Bind(req); err != nil {
			return err
		}
		if err := c.Validate(req); err != nil {
			return err
		}
		if req.Username == viper.GetString("panel.username") && req.Password == viper.GetString("panel.password") {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)), //过期日期设置7天
			}).SignedString([]byte(strconv.Itoa(os.Getpid())))
			if err != nil {
				return err
			}
			c.SetCookie(&http.Cookie{
				Name:     "panel_token",
				Value:    token,
				HttpOnly: true,
			})
			return c.Redirect(302, "/admin/monitor")
		}
		return echo.ErrUnauthorized
	}
	return echo.ErrMethodNotAllowed
}
