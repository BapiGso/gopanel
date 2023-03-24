package core

import (
	"crypto/sha1"
	"encoding/hex"
	"github.com/BapiGso/gopanel/core/monitor"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"net/http"
)

type loginReq struct {
	Username string `xml:"user" json:"user" form:"user" query:"user"`
	Password string `xml:"pwd" json:"pwd" form:"pwd" query:"pwd"`
}

func warning(c echo.Context) error {
	return c.Render(http.StatusOK, "warning.template", nil)
}

func loginGet(c echo.Context) error {
	sess, _ := session.Get("session", c)
	if sess.Values["isLogin"] == true {
		return c.Redirect(302, "/admin/home")
	}
	return c.Render(http.StatusOK, "login.template", nil)
}

func loginPost(c echo.Context) error {
	req := new(loginReq)
	//调用echo.Context的Bind函数将请求参数和User对象进行绑定。
	if err := c.Bind(req); err != nil {
		return c.String(200, "表单提交错误")
	}
	sess, _ := session.Get("session", c)
	sess.Values["isLogin"] = true
	//data := queryLogin()
	//if data.usr == req.Username && data.pwd == hash(req.Password+data.Salt) {
	//	sess.Values["isLogin"] = true
	//}
	sess.Save(c.Request(), c.Response())
	return c.Redirect(302, "/admin/home")
}

func home(c echo.Context) error {
	data := struct {
		Info *monitor.Monitor
	}{
		monitor.M,
	}
	return c.Render(200, "home.template", data)
}

func site(c echo.Context) error {
	return c.Render(200, "site.template", nil)
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
