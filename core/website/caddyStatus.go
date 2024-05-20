package website

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"os"
)

func start() error {
	adapter := caddyfile.Adapter{
		ServerType: &httpcaddyfile.ServerType{},
	}
	file, err := os.ReadFile("./Caddyfile")
	if err != nil {
		return err
	}
	jsonConfig, _, err := adapter.Adapt(caddyfile.Format(file), nil)
	if err != nil {
		return err
	}
	if err := caddy.Load(jsonConfig, true); err != nil {
		return err
	}
	return nil
}
