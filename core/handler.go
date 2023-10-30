package core

import (
	"panel/core/monitor"

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
