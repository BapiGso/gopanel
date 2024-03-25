package core

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
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
		log.Println("Unable to locate the configuration file. Creating new one.")
		// function to generate random string
		generateRandomString := func(n int) string {
			bytes := make([]byte, n)
			_, _ = rand.Read(bytes)
			return hex.EncodeToString(bytes)
		}
		// generate random strings
		randomPath := generateRandomString(4)
		randomUser := generateRandomString(6)
		randomPassword := generateRandomString(6)

		// set config
		viper.Set("port", ":8080")
		viper.Set("path", randomPath)
		viper.Set("username", randomUser)
		viper.Set("password", randomPassword)

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
