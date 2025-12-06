package frp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// 合并引用
	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/server"

	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

var (
	svr *server.Service
	cli *client.Service
)

// Index 处理统一的入口 /admin/frp
func Index(c echo.Context) error {
	serviceType := c.QueryParam("type") // "frps" or "frpc"

	switch c.Request().Method {
	case "POST":
		action := c.QueryParam("status") // "start", "enable", "stop"

		if action == "start" {
			if serviceType == "frps" {
				if err := runFRPSServer(); err != nil {
					return c.String(500, err.Error())
				}
			} else if serviceType == "frpc" {
				// FRPC start is async in original code
				go func() {
					if err := runFRPCClient(); err != nil {
						fmt.Printf("FRPC Start Error: %v\n", err)
					}
				}()
				// Give it a moment to try starting
				time.Sleep(500 * time.Millisecond)
			}
			return c.String(200, "Service started successfully")
		}

		if action == "enable" {
			configKey := fmt.Sprintf("enable.%s", serviceType)
			currentVal := viper.GetBool(configKey)
			viper.Set(configKey, !currentVal)

			if err := viper.WriteConfig(); err != nil {
				return c.String(500, "Failed to write config: "+err.Error())
			}
			return c.String(200, fmt.Sprintf("%s auto-boot toggled", serviceType))
		}

		// Optional: Stop logic (Complex for FRP, usually requires restart)
		// if action == "stop" { ... }

		return c.String(200, "success")

	case "PUT":
		// Save Config File
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}

		filename := ""
		if serviceType == "frps" {
			filename = "gopanel_frps.conf"
		} else if serviceType == "frpc" {
			filename = "gopanel_frpc.conf"
		} else {
			return c.String(400, "Invalid type")
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return err
		}
		return c.String(200, "Configuration saved successfully")

	case "GET":
		// Read both configs to render the merged page
		frpsFile, _ := os.ReadFile("gopanel_frps.conf")
		frpcFile, _ := os.ReadFile("gopanel_frpc.conf")

		return c.Render(http.StatusOK, "frp.template", map[string]any{
			"frpsConfig": string(frpsFile),
			"frpcConfig": string(frpcFile),
			"frpsEnable": viper.GetBool("enable.frps"),
			"frpcEnable": viper.GetBool("enable.frpc"),
		})
	}

	return echo.ErrMethodNotAllowed
}

// --- Service Runners ---

func runFRPSServer() error {
	// 如果已经运行，先尝试关闭 (简单处理，视FRP版本支持情况而定)
	if svr != nil {
		svr.Close()
	}

	cfg, _, err := config.LoadServerConfig("gopanel_frps.conf", false)
	if err != nil {
		return err
	}
	if _, err := validation.ValidateServerConfig(cfg); err != nil {
		return err
	}

	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	svr, err = server.NewService(cfg)
	if err != nil {
		return err
	}

	go svr.Run(context.Background())
	return nil
}

func runFRPCClient() error {
	// 如果已经运行，尝试优雅关闭
	if cli != nil {
		cli.GracefulClose(500 * time.Millisecond)
	}

	cfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig("gopanel_frpc.conf", false)
	warning, err := validation.ValidateAllClientConfig(cfg, proxyCfgs, visitorCfgs)
	if warning != nil {
		fmt.Printf("WARNING: %v\n", warning)
	}
	if err != nil {
		return err
	}

	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	cli, err = client.NewService(client.ServiceOptions{
		Common:      cfg,
		ProxyCfgs:   proxyCfgs,
		VisitorCfgs: visitorCfgs,
	})
	if err != nil {
		return err
	}

	shouldGracefulClose := cfg.Transport.Protocol == "kcp" || cfg.Transport.Protocol == "quic"
	if shouldGracefulClose {
		go handleTermSignal(cli)
	}

	return cli.Run(context.Background())
}

func handleTermSignal(svr *client.Service) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	svr.GracefulClose(500 * time.Millisecond)
}

// --- Initialization ---

func init() {
	// 1. 初始化 FRPS 默认配置
	frpsPath := "gopanel_frps.conf"
	if _, err := os.Stat(frpsPath); os.IsNotExist(err) {
		content := `# frps.conf
bindPort = 7000
auth.token = "public"
webServer.addr = "0.0.0.0"
webServer.port = 7500
webServer.user = "admin"
webServer.password = "admin"
`
		_ = os.WriteFile(frpsPath, []byte(content), 0644)
	}

	// 2. 初始化 FRPC 默认配置
	frpcPath := "gopanel_frpc.conf"
	if _, err := os.Stat(frpcPath); os.IsNotExist(err) {
		content := `# frpc.conf
serverAddr = "0.0.0.0"
serverPort = 7000
auth.token = "public"
[[proxies]]
name = "demo_tcp"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000
`
		_ = os.WriteFile(frpcPath, []byte(content), 0644)
	}

	// 3. 异步启动逻辑
	go func() {
		// 稍微延迟等待 Viper 加载完毕
		time.Sleep(3 * time.Second)

		if viper.GetBool("enable.frps") {
			fmt.Println("[FRPS] Auto-starting...")
			if err := runFRPSServer(); err != nil {
				fmt.Printf("[FRPS] Start failed: %v\n", err)
			}
		}

		if viper.GetBool("enable.frpc") {
			fmt.Println("[FRPC] Auto-starting...")
			if err := runFRPCClient(); err != nil {
				fmt.Printf("[FRPC] Start failed: %v\n", err)
			}
		}
	}()
}
