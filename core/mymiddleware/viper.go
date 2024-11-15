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
	showPanelAddr()
	fmt.Printf("Panel Port: %s\n", viper.GetString("panel.port"))
	fmt.Printf("Panel Path: %s\n", viper.GetString("panel.path"))
	fmt.Printf("Panel Username: %s\n", viper.GetString("panel.username"))
	fmt.Printf("Panel Password: %s\n", viper.GetString("panel.password"))
}

func showPanelAddr() {
	// Attempt to show IP address
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		} else if ipNet.IP.To4() != nil && ipNet.IP.IsGlobalUnicast() {
			//ip := ipNet.IP.To4()
			//if ip[0] == 10 || (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) || (ip[0] == 192 && ip[1] == 168) {
			//	continue
			//}
			fmt.Printf("gopanel started on https://%v%v/%v\n", ipNet.IP, viper.GetString("panel.port"), viper.GetString("panel.path"))
		} else if ipNet.IP.To16() != nil && ipNet.IP.IsGlobalUnicast() {
			// Check for IPv6 unicast addresses
			fmt.Printf("gopanel started on https://[%v]%v/%v\n", ipNet.IP, viper.GetString("panel.port"), viper.GetString("panel.path"))
		}
	}
}
