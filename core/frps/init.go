package frps

import (
	"os"
)

func init() {
	filePath := "frps.conf"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		content := `
# frps.conf
bindPort = 7000 				# 服务端与客户端通信端口
auth.token = "public" 			# 身份验证令牌，frpc要与frps一致
# Server Dashboard，可以查看frp服务状态以及统计信息
webServer.addr = "0.0.0.0"		# 后台管理地址
webServer.port = 7500 			# 后台管理端口
webServer.user = "admin"		# 后台登录用户名
webServer.password = "admin"	# 后台登录密码
`
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}
}
