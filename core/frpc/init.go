package frpc

import (
	"github.com/spf13/viper"
	"os"
	"time"
)

func init() {
	filePath := "gopanel_frpc.conf"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		content :=
			`# frpc.conf
serverAddr = "0.0.0.0"
serverPort = 7000
auth.token = ""


[[proxies]]
name = "proxy_name"
type = "tcp"
localIP = "127.0.0.1"
localPort = 8848
remotePort = 8848
`
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}
	go func() {
		time.Sleep(3 * time.Second)
		if viper.GetBool("enable.frpc") {
			_ = RunFRPCClient()
		}
	}()
}
