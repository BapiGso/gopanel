package core

import (
	"embed"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"gopanel/assets"
)

type Core struct {
	assetsFS *embed.FS  //主题所在文件夹
	e        *echo.Echo //后台框架
	_        *viper.Viper
	// 邮件提醒
}

func New() (c *Core) {
	c = &Core{}
	c.assetsFS = &assets.Assets
	c.e = echo.New()
	c.e.HideBanner = true
	return c
}
