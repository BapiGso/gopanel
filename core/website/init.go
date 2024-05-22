package website

import (
	"github.com/spf13/viper"
	"os"
	"time"
)

func init() {
	filePath := "./Caddyfile"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		content := "{\nadmin off\n}\n\n:2015\nrespond \"Hello, world!\""
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}
	go func() {
		time.Sleep(time.Second)
		if viper.Get("caddyEnable").(bool) {
			_ = start()
		}
	}()
}
