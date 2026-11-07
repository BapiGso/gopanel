package website

import (
	"gopanel/core/config"
	"gopanel/core/storage"
	"path/filepath"
)

func Initialize(directory string) error {
	caddyFile = &storage.File{Path: filepath.Join(directory, "gopanel_Caddyfile")}
	content :=
		`{
admin off
}

:2015
respond "Hello, world!"
`
	return caddyFile.Ensure([]byte(content))
}

func Autostart() error {
	if config.Snapshot().Enable.Caddy {
		return caddyStart()
	}
	return nil
}
