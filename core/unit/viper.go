package unit

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/spf13/viper"
)

// generate random strings
func generateRandomString(n int) string {
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func init() {
	viper.SetConfigName("config")                // name of config file (without extension)
	viper.SetConfigType("json")                  // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")                     // optionally look for config in the working directory
	if err := viper.ReadInConfig(); err != nil { // Handle errors reading the config file
		viper.Set("panel", map[string]string{
			"port":     ":8080",
			"path":     generateRandomString(4),
			"username": generateRandomString(6),
			"password": generateRandomString(6),
		})
		viper.Set("webdav", map[string]any{
			"enable":   false,
			"username": generateRandomString(3),
			"password": generateRandomString(3),
		})
		viper.Set("enable", map[string]any{
			"caddy": false,
			"frps":  false,
		})
		fmt.Printf("Panel Port: %s\n", viper.GetString("panel.port"))
		fmt.Printf("Panel Path: %s\n", viper.GetString("panel.path"))
		fmt.Printf("Panel Username: %s\n", viper.GetString("panel.username"))
		fmt.Printf("Panel Password: %s\n", viper.GetString("panel.password"))
		if err = viper.WriteConfigAs("config.json"); err != nil {
			fmt.Printf("Unable to create configuration file: %v", err)
		}
	}
}
