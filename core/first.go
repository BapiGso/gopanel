package core

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	"math/rand"
	_ "modernc.org/sqlite"
	"os"
	"time"
)

const congratulation = "安装成功" +
	"==========================" +
	"Gopanel:http://localhost:%v/%v" +
	"Username:%v" +
	"Password:%v" +
	"如果你忘记了用户密码，请执行./gopanel -r"

type registerUsr struct {
	port string `db:"path"`
	path string `db:"path"`
	usr  string `db:"usr"`
	pwd  string `db:"pwd"`
}

func (c *Core) First() {
	_, err := os.Stat("./panel.db")
	if os.IsNotExist(err) {
		fmt.Println("看起来您是第一次使用面板，正在初始化数据库。。。")
		c.Db, err = sqlx.Connect("sqlite", "panel.db")
		if err != nil {
			log.Fatalf("创建数据库失败，请检查读写权限%v\n", err)
		}
		if err = c.initDb(); err != nil {
			log.Fatalf("读取sql文件失败，请检查读写权限%v\n", err)
		}
		if conf, err := c.registerUsr(); err != nil {
			log.Print(congratulation, conf.path, conf.usr, conf.pwd)
		}
	}
	c.Db, err = sqlx.Connect("sqlite", "panel.db")
	return
}

// 初始化数据库结构
func (c *Core) initDb() error {
	sqlTable, err := c.AssetsFS.ReadFile("panel.sql")
	_, err = c.Db.Exec(string(sqlTable))
	return err
}

// 初始化数据库结构
func (c *Core) registerUsr() (*registerUsr, error) {
	data := &registerUsr{
		port: "8848",
		path: string(randStr()),
		usr:  string(randStr()),
		pwd:  string(randStr()),
	}
	_, err := c.Db.NamedExec("INSERT INTO conf (port, path, usr, pwd) VALUES (:port, :path, :usr, :pwd)", &data)
	return data, err
}

func randStr() []byte {
	rand.Seed(time.Now().UnixNano())
	// 可选的字符集合
	charSet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	// 生成随机字符串
	randomString := make([]byte, 10)
	for i := 0; i < 10; i++ {
		randomString[i] = charSet[rand.Intn(len(charSet))]
	}
	return randomString
}
