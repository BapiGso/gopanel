package website

import (
	"github.com/spf13/viper"
	"os"
)

func init() {
	filePath := "Caddyfile"
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
	if viper.GetBool("enable.caddy") {
		_ = caddyStart()
	}
}
