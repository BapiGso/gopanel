package website

import (
	"gopanel/core/config"
	"os"
)

func init() {
	filePath := "gopanel_Caddyfile"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		content :=
			`{
admin off
}

:2015
respond "Hello, world!"
`
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}
	if config.Bool("enable.caddy") {
		_ = caddyStart()
	}
}
