package mymiddleware

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log/slog"
	"os"
)

func init() {
	l := slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelInfo,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			},
		),
	)
	slog.SetDefault(l)
}

var Slog = middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
	LogMethod:   true,
	LogURI:      true,
	LogStatus:   true,
	LogRemoteIP: true,
	LogError:    true,
	LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
		slog.LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprintf("err=%v", v.Error),
			slog.String("Method", v.Method),
			slog.String("Url", v.URI),
			slog.String("IP", v.RemoteIP),
			slog.Int("Status", v.Status),
		)
		return nil
	},
})
