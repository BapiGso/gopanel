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
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
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
				"caddy": false,
				"frps":  false,
			},
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Printf("json marshal: %v\n", err)
		}

		if err := os.WriteFile("config.json", jsonData, 0644); err != nil {
			fmt.Printf("write file: %v\n", err)
		}
	}
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("read config: %v\n", err)
	}
	//todo show ip addr
	fmt.Printf("Panel Port: %s\n", viper.GetString("panel.port"))
	fmt.Printf("Panel Path: %s\n", viper.GetString("panel.path"))
	fmt.Printf("Panel Username: %s\n", viper.GetString("panel.username"))
	fmt.Printf("Panel Password: %s\n", viper.GetString("panel.password"))
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		// 检查地址类型是否为 IP 网络地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// 检查是否为 IPv4 地址
			return ipnet.IP.String(), nil
		}
	}

	return "", fmt.Errorf("no IP address found")
}
