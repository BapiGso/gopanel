package mymiddleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"net"
	"os"
)

// generate random strings
func generateRandomString(n int) string {
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func init() {
	if _, err := os.Stat("gopanel_config.json"); os.IsNotExist(err) {
		data := map[string]any{
			"panel": map[string]any{
				"port":     ":8443",
				"path":     generateRandomString(4),
				"username": generateRandomString(6),
				"password": generateRandomString(6),
			},
			"webdav": map[string]any{
				"enable":   false,
				"username": generateRandomString(3),
				"password": generateRandomString(3),
			},
			"enable": map[string]any{
				"caddy":     false,
				"frps":      false,
				"frpc":      false,
				"headscale": false,
			},
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Printf("json marshal: %v\n", err)
		}

		if err := os.WriteFile("gopanel_config.json", jsonData, 0644); err != nil {
			fmt.Printf("write file: %v\n", err)
		}
	}
	viper.SetConfigName("gopanel_config") // name of config file (without extension)
	viper.SetConfigType("json")           // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")              // optionally look for config in the working directory

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("read config: %v\n", err)
	}
	ColorGreen := "\x1b[32m" // 绿色开始
	ColorReset := "\x1b[0m"  // 重置颜色
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		} else if ipNet.IP.To4() != nil && ipNet.IP.IsGlobalUnicast() {
			fmt.Printf("gopanel started on https://%v%v/%v\n", ipNet.IP, viper.GetString("panel.port"), viper.GetString("panel.path"))
		} else if ipNet.IP.To16() != nil && ipNet.IP.IsGlobalUnicast() {
			fmt.Printf("gopanel started on https://[%v]%v/%v\n", ipNet.IP, viper.GetString("panel.port"), viper.GetString("panel.path"))
		}
	}
	fmt.Printf("Panel Port: %s%s%s\n", ColorGreen, viper.GetString("panel.port"), ColorReset)
	fmt.Printf("Panel Path: %s%s%s\n", ColorGreen, viper.GetString("panel.path"), ColorReset)
	fmt.Printf("Panel Username: %s%s%s\n", ColorGreen, viper.GetString("panel.username"), ColorReset)
	fmt.Printf("Panel Password: %s%s%s\n", ColorGreen, viper.GetString("panel.password"), ColorReset)
}
