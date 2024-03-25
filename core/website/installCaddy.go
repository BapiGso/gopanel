package website

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

func installCaddy() {
	url := fmt.Sprintf("https://caddyserver.com/api/download?os=%s&arch=%s", runtime.GOOS, runtime.GOARCH)
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("下载Caddy失败: ", err)
	}
	defer resp.Body.Close()
	// 创建caddy文件
	out, err := os.Create("caddy")
	if err != nil {
		fmt.Println("创建Caddy文件失败: ", err)
	}
	defer out.Close()
	// 将响应体的内容写入文件
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("写入Caddy文件失败: ", err)
	}
	fmt.Println("下载Caddy成功.")

}
