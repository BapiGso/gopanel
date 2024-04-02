package website

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2"
	//_ "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	//_ "github.com/caddyserver/caddy/v2/modules/caddyevents"
	//_ "github.com/caddyserver/caddy/v2/modules/caddyevents/eventsconfig"
	//_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/standard"
	//_ "github.com/caddyserver/caddy/v2/modules/caddypki"
	//_ "github.com/caddyserver/caddy/v2/modules/caddypki/acmeserver"
	//_ "github.com/caddyserver/caddy/v2/modules/caddytls"
	//_ "github.com/caddyserver/caddy/v2/modules/caddytls/distributedstek"
	//_ "github.com/caddyserver/caddy/v2/modules/caddytls/standardstek"
	//_ "github.com/caddyserver/caddy/v2/modules/filestorage"
	//_ "github.com/caddyserver/caddy/v2/modules/logging"
	"github.com/fsnotify/fsnotify"
	"time"

	//_ "github.com/caddyserver/caddy/v2/modules/caddyfs"
	"log/slog"

	//_ "github.com/caddyserver/caddy/v2/modules/metrics"
	_ "github.com/caddyserver/caddy/v2/modules/standard" // required for initializing standard HTTP modules
	"github.com/spf13/viper"
)

var lastRead = time.Now()

var caddyConfig = func() *viper.Viper {
	//caddyfile.Format()
	cv := viper.New()
	cv.SetConfigName("caddyConfig") // 设置配置文件名 (不需要带后缀)
	cv.SetConfigType("json")        // 设置配置文件类型
	cv.AddConfigPath(".")           // 设置配置文件路径
	if err := cv.ReadInConfig(); err != nil {
		//https://caddyserver.com/docs/json 主要是配apps字段
		cv.Set("logging.logs.access_log.writer", map[string]string{
			"output":   "file",
			"filename": "/access.log",
		})
		cv.Set("apps.http.servers.example.listen", []string{":2015"})
		cv.Set("apps.http.servers.example.routes", []any{
			map[string]any{
				"match": []any{
					map[string]any{
						"host": []string{"example.com"}},
				},
				"handle": []any{
					map[string]any{
						"handler": "static_response",
						"body":    "Hello, world!",
					},
				},
			},
		})
		if err = cv.WriteConfigAs("caddyConfig.json"); err != nil {
			slog.Error("Unable to create caddy configuration file.", err)
		}
	} // 读取配置数据
	cv.OnConfigChange(func(e fsnotify.Event) {
		if time.Since(lastRead) < time.Second {
			return
		}
		lastRead = time.Now()
		if err := caddyStart(convertJSON(cv.Get("logging.logs.access_log.writer")), convertJSON(cv.Get("apps.http"))); err != nil {
			slog.Error("caddy start error:", err)
		}
		// 这里可以放置你的代码来处理配置更改
	})
	cv.WatchConfig()
	return cv
}()

func caddyStart(log, http []byte) error {
	if err := caddy.Stop(); err != nil {
		return err
	}
	//if err := caddy.Validate(&conf); err != nil {
	//	return err
	//}
	return caddy.Run(&caddy.Config{
		Admin: &caddy.AdminConfig{Disabled: true},
		Logging: &caddy.Logging{
			Sink: nil,
			Logs: map[string]*caddy.CustomLog{
				"access": {
					BaseLog: caddy.BaseLog{
						WriterRaw: log,
					},
				},
			},
		},
		StorageRaw: nil,
		AppsRaw: map[string]json.RawMessage{
			"http": http,
		},
	})
}

func caddyStop() error {
	return caddy.Stop()
}
