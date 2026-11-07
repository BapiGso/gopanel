package core

import (
	"embed"
	"github.com/labstack/echo/v5"
	"gopanel/assets"
	"gopanel/core/config"
	"gopanel/core/monitor"
)

type Core struct {
	assetsFS     *embed.FS  //主题所在文件夹
	e            *echo.Echo //后台框架
	monitoring   *monitor.Service
	startupError error
	// 邮件提醒
}

func New() (c *Core) {
	c = &Core{}
	c.assetsFS = &assets.Assets
	c.e = echo.New()
	settings, err := monitor.LoadConfig(config.Directory())
	if err != nil {
		c.startupError = err
		return c
	}
	c.monitoring, c.startupError = monitor.NewService(settings, nil, c.e.Logger)
	//c.e.HideBanner = true
	return c
}
