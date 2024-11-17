package frps

import (
	"github.com/spf13/viper"
	"os"
	"time"
)

func init() {
	filePath := "gopanel_frps.conf"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		content :=
			`# frps.conf
bindPort = 7000 				# Port for communication between server and client
auth.token = "public" 			# Authentication token, must match between frpc and frps

# Server Dashboard, can view the status and statistics of the frp service
webServer.addr = "0.0.0.0"		# Background management address
webServer.port = 7500 			# Background management port
webServer.user = "admin"		# Background login username
webServer.password = "admin"	# Background login password
`
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}
	go func() {
		time.Sleep(3 * time.Second)
		if viper.GetBool("enable.frps") {
			_ = runFRPSServer()
		}
	}()

}
