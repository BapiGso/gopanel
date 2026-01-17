package mymiddleware

import (
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"net/http"
	"os"
	"strconv"
)

var JWT, _ = echojwt.Config{
	ErrorHandler: func(c *echo.Context, err error) error {
		return c.Render(http.StatusTeapot, "warning.template", map[string]string{
			"message": err.Error(),
			"ip":      c.RealIP(),
		})
	},
	SigningKey:  []byte(strconv.Itoa(os.Getpid())),
	TokenLookup: "cookie:panel_token",
	Skipper:     func(c *echo.Context) bool { return os.Getenv("GOPANEL_DEBUG") == "1" },
}.ToMiddleware()
