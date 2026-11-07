package website

import (
	"encoding/json"
	"net/http"
	"sync"

	"gopanel/core/config"
	"gopanel/core/httpapi"
	"gopanel/core/storage"

	_ "github.com/caddy-dns/cloudflare"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/standard" //不导入这个不行

	_ "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/caddyevents"
	_ "github.com/caddyserver/caddy/v2/modules/caddyevents/eventsconfig"
	_ "github.com/caddyserver/caddy/v2/modules/caddyfs"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/standard"
	_ "github.com/caddyserver/caddy/v2/modules/caddypki"
	_ "github.com/caddyserver/caddy/v2/modules/caddypki/acmeserver"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls/distributedstek"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls/standardstek"
	_ "github.com/caddyserver/caddy/v2/modules/filestorage"
	_ "github.com/caddyserver/caddy/v2/modules/logging"
	_ "github.com/caddyserver/caddy/v2/modules/metrics"
)

var (
	caddyFile     = &storage.File{Path: "gopanel_Caddyfile"}
	caddyMutex    sync.Mutex
	caddyRunning  bool
	caddyRevision string
)

func adaptConfig(content string) ([]byte, error) {
	adapter := caddyfile.Adapter{
		ServerType: &httpcaddyfile.ServerType{},
	}
	jsonConfig, _, err := adapter.Adapt(caddyfile.Format([]byte(content)), nil)
	if err != nil {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", err.Error())
	}
	var configuration caddy.Config
	if err := json.Unmarshal(jsonConfig, &configuration); err != nil {
		return nil, err
	}
	persist := false
	configuration.Admin = &caddy.AdminConfig{Disabled: true, Config: &caddy.ConfigSettings{Persist: &persist}}
	return json.Marshal(configuration)
}

func caddyStart() error {
	caddyMutex.Lock()
	defer caddyMutex.Unlock()
	document, err := caddyFile.Read()
	if err != nil {
		return err
	}
	jsonConfig, err := adaptConfig(document.Content)
	if err != nil {
		return err
	}
	if err := caddy.Load(jsonConfig, true); err != nil {
		return httpapi.NewError(http.StatusUnprocessableEntity, "apply_failed", err.Error())
	}
	caddyRunning = true
	caddyRevision = document.Revision
	return nil
}

func caddyStop() error {
	caddyMutex.Lock()
	defer caddyMutex.Unlock()
	if err := caddy.Stop(); err != nil {
		return err
	}
	caddyRunning = false
	caddyRevision = ""
	return nil
}

func status() (httpapi.ServiceStatus, error) {
	settings, err := config.Read()
	if err != nil {
		return httpapi.ServiceStatus{}, err
	}
	caddyMutex.Lock()
	defer caddyMutex.Unlock()
	document, err := caddyFile.Read()
	if err != nil {
		return httpapi.ServiceStatus{}, err
	}
	state := "stopped"
	if caddyRunning {
		state = "running"
	}
	return httpapi.ServiceStatus{
		State: state, Autostart: settings.Config.Enable.Caddy, AppliedRevision: caddyRevision,
		SettingsRevision: settings.Revision,
		PendingChanges:   document.Revision != caddyRevision,
	}, nil
}
