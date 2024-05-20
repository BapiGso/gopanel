package core

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"panel/assets"
	"panel/core/website"
	"time"
)

type Core struct {
	assetsFS *embed.FS         //主题所在文件夹
	e        *echo.Echo        //后台框架
	s        *gocron.Scheduler //任务计划
	// 邮件提醒
}

func New() (c *Core) {
	viper.SetConfigName("config")                // name of config file (without extension)
	viper.SetConfigType("json")                  // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")                     // optionally look for config in the working directory
	if err := viper.ReadInConfig(); err != nil { // Handle errors reading the config file
		// function to generate random string
		generateRandomString := func(n int) string {
			bytes := make([]byte, n)
			_, _ = rand.Read(bytes)
			return hex.EncodeToString(bytes)
		}
		// generate random strings
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
		viper.Set("caddyEnable", false)
		fmt.Printf("Panel Port: %s\n", viper.GetString("panel.port"))
		fmt.Printf("Panel Path: %s\n", viper.GetString("panel.path"))
		fmt.Printf("Panel Username: %s\n", viper.GetString("panel.username"))
		fmt.Printf("Panel Password: %s\n", viper.GetString("panel.password"))
		// save the config file
		if err = viper.WriteConfigAs("config.json"); err != nil {
			fmt.Printf("Unable to create configuration file: %v", err)
		}
	}
	website.Init()
	c = &Core{}
	c.assetsFS = &assets.Assets
	c.e = echo.New()
	c.e.HideBanner = true
	c.s = gocron.NewScheduler(time.UTC)
	return c
}
