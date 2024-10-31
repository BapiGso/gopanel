package frps

import (
	"context"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/server"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
)

var svr *server.Service

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "POST":
		if c.QueryParam("status") == "start" {
			if err := RunFRPSServer(); err != nil {
				return err
			}
		}
		if c.QueryParam("status") == "stop" {
			return svr.Close()
		}
		if c.QueryParam("status") == "enable" {
			viper.Set("enable.frps", !viper.Get("enable.frps").(bool))
		}
		return c.JSON(200, "success")

	case "PUT":
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile("gopanel_frps.conf", data, 0644); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "GET":
		file, err := os.ReadFile("gopanel_frps.conf")
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "frps.template", map[string]any{
			"frpsConfig": string(file),
			"frpsEnable": viper.Get("enable.frps").(bool),
		})
	}

	return echo.ErrMethodNotAllowed
}

func RunFRPSServer() error {
	//读取文件转为配置
	cfg, _, err := config.LoadServerConfig("gopanel_frps.conf", strictConfigMode)
	if err != nil {
		return err
	}
	//校验配置
	if _, err := validation.ValidateServerConfig(cfg); err != nil {
		return err
	}
	// 初始化日志
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	// 创建并启动服务器
	svr, err = server.NewService(cfg)
	if err != nil {
		return err
	}
	go svr.Run(context.Background())
	return nil
}
