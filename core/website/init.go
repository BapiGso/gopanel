package website

import (
	"github.com/spf13/viper"
	"os"
)

// Init 这个函数在core.New中手动执行，因为依赖viper的初始化
func Init() {
	filePath := "./Caddyfile"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		content := ":2015\n\nrespond \"Hello, world!\""
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}
	if viper.Get("caddyEnable").(bool) {
		_ = start()
	}
}
