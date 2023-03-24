package core

import (
	"embed"
	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"gopanel/assets"
	"text/template"
	"time"
)

type (
	Core struct {
		CommandLineArgs BindFlag          //命令行参数
		Db              *sqlx.DB          //数据库
		AssetsFS        *embed.FS         //主题所在文件夹
		E               *echo.Echo        //后台框架
		S               *gocron.Scheduler //任务计划
		// 邮件提醒
	}

	TemplateRender struct {
		Template *template.Template //渲染模板
	}
)

const (
	banner = `
 ______     ______     ______   ______     __   __     ______     __        
/\  ___\   /\  __ \   /\  == \ /\  __ \   /\ "-.\ \   /\  ___\   /\ \       
\ \ \__ \  \ \ \/\ \  \ \  _-/ \ \  __ \  \ \ \-.  \  \ \  __\   \ \ \____  
 \ \_____\  \ \_____\  \ \_\    \ \_\ \_\  \ \_\\"\_\  \ \_____\  \ \_____\ 
  \/_____/   \/_____/   \/_/     \/_/\/_/   \/_/ \/_/   \/_____/   \/_____/

____________________________________O/_______
                                    O\
%s
`
)

func New() (c *Core) {
	c = &Core{}
	c.AssetsFS = &assets.Assets
	c.E = echo.New()
	c.S = gocron.NewScheduler(time.UTC)
	return c
}
