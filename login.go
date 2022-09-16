package main

import (
	"crypto/sha1"
	"encoding/hex"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"net/http"
)

type loginReq struct {
	Username string `xml:"user" json:"user" form:"user" query:"user"`
	Password string `xml:"pwd" json:"pwd" form:"pwd" query:"pwd"`
	Illsions string `xml:"illsions" json:"illsions" form:"illsions" query:"illsions"`
}

type loginSql struct {
	Username string
	Password string
	Salt     string
}

func queryLogin() *loginSql {
	data := new(loginSql)
	_ = db.QueryRow("SELECT username,password,salt FROM user WHERE id='1'").Scan(&data.Username, &data.Password, &data.Salt)
	return data
}

func LoginGet(c echo.Context) error {
	sess, _ := session.Get("smoesession", c)
	if sess.Values["isLogin"] == true {
		return c.Redirect(302, "/home")
	}
	return c.Render(http.StatusOK, "nologin.template", nil)
}

// todo 防爆破
// todo monitor
func LoginPost(c echo.Context) error {
	req := new(loginReq)
	//调用echo.Context的Bind函数将请求参数和User对象进行绑定。
	if err := c.Bind(req); err != nil {
		return c.String(200, "表单提交错误")
	}
	sess, _ := session.Get("panelsession", c)
	//TODO 发邮件提醒和防爆破
	data := queryLogin()
	if data.Username == req.Username && data.Password == hash(req.Password+data.Salt) {
		sess.Values["isLogin"] = true
	} else {
		sess.Values["isLogin"] = false
	}
	sess.Save(c.Request(), c.Response())
	return c.Redirect(302, "/admin")
}

func hash(passwd string) string {
	h := sha1.New() // md5加密类似md5.New()
	//写入要处理的字节。如果是一个字符串，需要使用[]byte(s) 来强制转换成字节数组。
	h.Write([]byte(passwd))
	bs := h.Sum(nil)
	h.Reset()
	passwdhash := hex.EncodeToString(bs) //转16进制
	return passwdhash
}
