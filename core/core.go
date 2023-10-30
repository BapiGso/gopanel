package core

import (
	"embed"
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"panel/assets"
	"panel/core/unit"
	"time"
)

type Core struct {
	Conf            *conf             //存储配置文件
	CommandLineArgs BindFlag          //命令行参数
	db              *sqlx.DB          //数据库
	assetsFS        *embed.FS         //主题所在文件夹
	e               *echo.Echo        //后台框架
	s               *gocron.Scheduler //任务计划
	// 邮件提醒
}

type conf struct {
	LoginPath string
	JWTKey    []byte
	User      []byte
	PassWord  []byte
}

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
	c.Conf = &conf{
		LoginPath: unit.RandStr(10),
		JWTKey:    []byte(unit.RandStr(12)),
		User:      []byte(unit.RandStr(8)),
		PassWord:  []byte(unit.RandStr(8)),
	}
	fmt.Println(c.Conf)
	c.assetsFS = &assets.Assets
	c.e = echo.New()
	c.s = gocron.NewScheduler(time.UTC)
	return c
}
