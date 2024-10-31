package frpc

import (
	"context"
	"fmt"
	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//var svr *server.Service

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "POST":
		if c.QueryParam("status") == "start" {
			errCh := make(chan error)
			go func() {
				errCh <- RunFRPCClient()
			}()
			select {
			case err := <-errCh:
				if err != nil {
					return err
				}
			case <-time.After(time.Second):
				return c.JSON(200, "success")
			}
		}
		if c.QueryParam("status") == "enable" {
			viper.Set("enable.frpc", !viper.Get("enable.frpc").(bool))
			if err := viper.WriteConfig(); err != nil {
				return err // 处理错误
			}
		}
		return c.JSON(200, "success")

	case "PUT":
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile("gopanel_frpc.conf", data, 0644); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "GET":
		file, err := os.ReadFile("gopanel_frpc.conf")
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "frpc.template", map[string]any{
			"frpcConfig": string(file),
			"frpcEnable": viper.Get("enable.frpc").(bool),
		})
	}

	return echo.ErrMethodNotAllowed
}

func RunFRPCClient() error {
	cfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig("gopanel_frpc.conf", strictConfigMode)
	warning, err := validation.ValidateAllClientConfig(cfg, proxyCfgs, visitorCfgs)
	if warning != nil {
		fmt.Printf("WARNING: %v\n", warning)
	}
	if err != nil {
		return err
	}
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)
	svr, err := client.NewService(client.ServiceOptions{
		Common:         cfg,
		ProxyCfgs:      proxyCfgs,
		VisitorCfgs:    visitorCfgs,
		ConfigFilePath: cfgFile,
	})
	if err != nil {
		return err
	}

	shouldGracefulClose := cfg.Transport.Protocol == "kcp" || cfg.Transport.Protocol == "quic"
	// Capture the exit signal if we use kcp or quic.
	if shouldGracefulClose {
		go handleTermSignal(svr)
	}
	return svr.Run(context.Background())
}

func handleTermSignal(svr *client.Service) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	svr.GracefulClose(500 * time.Millisecond)
}
