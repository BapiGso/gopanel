package main

import (
	"crypto/tls"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"github.com/gorilla/sessions"
	_ "github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	_ "modernc.org/sqlite"
	"net/http"
	_ "net/http/pprof"
	"text/template"
)

var db *sql.DB
var panelpath string

//go:embed template
var tempfile embed.FS

//go:embed shell
var shellfile embed.FS

func init() {
	checkDB()
	randompath()
}

func checkDB() {
	//不存在就创建数据库
	var err error
	db, err = sql.Open("sqlite", "panel.db")
	if err != nil {
		log.Fatalf("创建数据库失败，请检查读写权限%v\n", err)
	}
	//读取sql文件创建表
	sqlTable, err := ioutil.ReadFile("shell/table.sql")
	if err != nil {
		log.Fatalf("读取sql文件失败，请检查读写权限%v\n", err)
	}
	db.Exec(string(sqlTable))
}

func randompath() {
	letterBytes := "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	panelpath = "/" + string(b)
}

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func IsLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//登录页面不用这个中间件
		if c.Path() == panelpath {
			return next(c)
		}
		//后台页面没有cookie的全部跳去登录
		sess, _ := session.Get("session", c)
		if sess.Values["isLogin"] != true {
			return c.Redirect(http.StatusFound, panelpath)
		}
		return next(c)
	}
}

// todo auto tls
func main() {
	go http.ListenAndServe(":8080", nil)
	flag.Parse()
	e := echo.New()

	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.ParseFS(tempfile, "template/*.template")),
	}

	//e.Logger.SetLevel(log.DEBUG)
	//Secure防XSS，HSTS防中间人攻击
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		HSTSMaxAge:            31536000,
		HSTSPreloadEnabled:    true,
		HSTSExcludeSubdomains: true,
	}))

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	//e.AutoTLSManager.Cache = autocert.DirCache("/var/www/.cache")
	//echoV5更新时换成broitil编码
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 3,
	}))
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	e.Static("/assets", "./assets")
	e.GET("/", LoginGet)

	g := e.Group(panelpath)

	g.Use(IsLogin)
	g.GET("", LoginGet)
	g.POST("", LoginPost)
	//e.Start(*bind)
	go fmt.Println("panel path is:", panelpath)
	e.Logger.Fatal(e.Start(":8848"))
}

func customHTTPServer() {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `
			<h1>Welcome to Echo!</h1>
			<h3>TLS certificates automatically installed from Let's Encrypt :)</h3>
		`)
	})

	autoTLSManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		// Cache certificates to avoid issues with rate limits (https://letsencrypt.org/docs/rate-limits)
		Cache: autocert.DirCache("/var/www/.cache"),
		//HostPolicy: autocert.HostWhitelist("<DOMAIN>"),
	}
	s := http.Server{
		Addr:    ":443",
		Handler: e, // set Echo as handler
		TLSConfig: &tls.Config{
			//Certificates: nil, // <-- s.ListenAndServeTLS will populate this field
			GetCertificate: autoTLSManager.GetCertificate,
			NextProtos:     []string{acme.ALPNProto},
		},
		//ReadTimeout: 30 * time.Second, // use custom timeouts
	}
	if err := s.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		e.Logger.Fatal(err)
	}
}
