package config

import (
	"crypto/rand"
	"encoding/hex"
)

const FileName = "gopanel_config.json"
const SchemaVersion = 1

type Config struct {
	SchemaVersion int          `json:"schema_version" validate:"eq=1"`
	Panel         PanelConfig  `json:"panel"`
	WebDAV        WebDAVConfig `json:"webdav"`
	Enable        EnableConfig `json:"enable"`
}

type PanelConfig struct {
	Port     string `json:"port" validate:"required,max=200,listen_address"`
	Path     string `json:"path" validate:"required,max=200,panel_path"`
	Username string `json:"username" validate:"required,max=200"`
	Password string `json:"password" validate:"required,max=200"`
}

type WebDAVConfig struct {
	Enable   bool   `json:"enable"`
	Username string `json:"username" validate:"required_if=Enable true,max=200"`
	Password string `json:"password" validate:"required_if=Enable true,max=200"`
}

type EnableConfig struct {
	Caddy     bool `json:"caddy"`
	Frps      bool `json:"frps"`
	Frpc      bool `json:"frpc"`
	Headscale bool `json:"headscale"`
}

func Default() (Config, error) {
	lengths := []int{4, 6, 6, 3, 3}
	credentials := make([]string, len(lengths))
	for index, length := range lengths {
		value, err := randomString(length)
		if err != nil {
			return Config{}, err
		}
		credentials[index] = value
	}
	return Config{
		SchemaVersion: SchemaVersion,
		Panel: PanelConfig{
			Port:     ":8443",
			Path:     credentials[0],
			Username: credentials[1],
			Password: credentials[2],
		},
		WebDAV: WebDAVConfig{
			Enable:   false,
			Username: credentials[3],
			Password: credentials[4],
		},
		Enable: EnableConfig{
			Caddy:     false,
			Frps:      false,
			Frpc:      false,
			Headscale: false,
		},
	}, nil
}

func randomString(length int) (string, error) {
	content := make([]byte, length)
	if _, err := rand.Read(content); err != nil {
		return "", err
	}
	return hex.EncodeToString(content), nil
}
