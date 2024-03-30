package core

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"log"
	"panel/assets"
	"time"
)

type Core struct {
	assetsFS *embed.FS         //主题所在文件夹
	e        *echo.Echo        //后台框架
	s        *gocron.Scheduler //任务计划
	// 邮件提醒
}

func New() (c *Core) {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory

	if err := viper.ReadInConfig(); err != nil { // Handle errors reading the config file
		// function to generate random string
		generateRandomString := func(n int) string {
			bytes := make([]byte, n)
			_, _ = rand.Read(bytes)
			return hex.EncodeToString(bytes)
		}
		// generate random strings
		viper.Set("panel.port", ":8080")
		viper.Set("panel.path", generateRandomString(4))
		viper.Set("panel.username", generateRandomString(6))
		viper.Set("panel.password", generateRandomString(6))
		viper.Set("webdav.enable", false)
		viper.Set("webdav.username", generateRandomString(6))
		viper.Set("webdav.password", generateRandomString(6))
		fmt.Println("Initial configuration for the panel:")
		fmt.Printf("Panel Port: %s\n", viper.GetString("panel.port"))
		fmt.Printf("Panel Path: %s\n", viper.GetString("panel.path"))
		fmt.Printf("Panel Username: %s\n", viper.GetString("panel.username"))
		fmt.Printf("Panel Password: %s\n", viper.GetString("panel.password"))
		// save the config file
		if err = viper.WriteConfigAs("config.json"); err != nil {
			log.Fatalln("Unable to create configuration file.", err)
		}

	}
	c = &Core{}
	c.assetsFS = &assets.Assets
	c.e = echo.New()
	c.e.HideBanner = true
	c.s = gocron.NewScheduler(time.UTC)
	return c
}
