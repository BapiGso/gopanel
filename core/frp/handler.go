package frp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/storage"
)

var configFiles = map[string]*storage.File{
	"frps": {Path: "gopanel_frps.conf"},
	"frpc": {Path: "gopanel_frpc.conf"},
}

func Index(ctx *echo.Context) error {
	settings, err := config.Read()
	if err != nil {
		return err
	}
	serverConfig, err := configFiles["frps"].Read()
	if err != nil {
		return err
	}
	clientConfig, err := configFiles["frpc"].Read()
	if err != nil {
		return err
	}
	serverContent, err := json.Marshal(serverConfig.Content)
	if err != nil {
		return err
	}
	clientContent, err := json.Marshal(clientConfig.Content)
	if err != nil {
		return err
	}
	return ctx.Render(http.StatusOK, "frp.template", map[string]any{
		"frpsConfig": string(serverContent), "frpcConfig": string(clientContent),
		"frpsRevision": serverConfig.Revision, "frpcRevision": clientConfig.Revision,
		"frpsEnable": settings.Config.Enable.Frps, "frpcEnable": settings.Config.Enable.Frpc,
		"settingsRevision": settings.Revision,
	})
}

func Initialize(directory string) error {
	defaults := map[string]string{
		"frps": `bindPort = 7000
auth.token = "public"
webServer.addr = "0.0.0.0"
webServer.port = 7500
webServer.user = "admin"
webServer.password = "admin"
`,
		"frpc": `serverAddr = "0.0.0.0"
serverPort = 7000
auth.token = "public"
[[proxies]]
name = "demo_tcp"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000
`,
	}
	for name, content := range defaults {
		configFiles[name] = &storage.File{Path: filepath.Join(directory, "gopanel_"+name+".conf")}
		if err := configFiles[name].Ensure([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func Autostart() error {
	settings := config.Snapshot().Enable
	var failures []error
	for _, name := range []string{"frps", "frpc"} {
		enabled := settings.Frps
		if name == "frpc" {
			enabled = settings.Frpc
		}
		if enabled {
			if err := startService(name, false); err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", name, err))
			}
		}
	}
	return errors.Join(failures...)
}
