package core

import (
	"gopanel/core/cron"
	"gopanel/core/docker"
	"gopanel/core/file"
	"gopanel/core/firewall"
	"gopanel/core/frp"

	"gopanel/core/headscale"
	"gopanel/core/login"
	"gopanel/core/monitor"
	"gopanel/core/mymiddleware"
	"gopanel/core/security"
	"gopanel/core/term"
	"gopanel/core/webdav"
	"gopanel/core/website"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
)

func (c *Core) Route() {
	c.e.Validator = mymiddleware.DefaultValidator
	c.e.Renderer = mymiddleware.DefaultTemplateRender

	c.e.Use(middleware.RequestLogger())
	c.e.Use(middleware.Recover())
	c.e.Use(middleware.Gzip())

	c.e.HTTPErrorHandler = func(err error, c echo.Context) {
		c.JSON(400, err.Error())
	}
	//限制频率
	c.e.Any(viper.GetString("panel.path"), login.Login, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(3)))

	c.e.Match([]string{"GET", "HEAD", "POST", "OPTIONS", "PUT", "MKCOL",
		"DELETE", "PROPFIND", "PROPPATCH", "COPY", "MOVE", "REPORT",
		"LOCK", "UNLOCK"}, "/webdav*", webdav.WebDav)

	// 静态资源
	//c.e.StaticFS("/assets", c.assetsFS)
	c.e.Group("/assets", middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper: func(c echo.Context) bool {
			c.Response().Header().Set("Cache-Control", "public, max-age=86400")
			return false
		},
		Filesystem: http.FS(c.assetsFS),
	}))
	//用于PWA的路径重写
	c.e.Pre(middleware.Rewrite(map[string]string{
		"/manifest.webmanifest": "/assets/manifest.webmanifest",
		"/sw.js":                "/assets/js/sw.js",
	}))
	// 后台路由
	admin := c.e.Group("/admin")
	admin.Use(mymiddleware.JWT)
	admin.GET("/monitor", monitor.Index)
	admin.Any("/website", website.Index)
	admin.GET("/file", file.Index)
	admin.Any("/file/process", file.Process)
	admin.Any("/webdav", webdav.Index)
	admin.GET("/term", term.Index)
	admin.POST("/term", term.CreateTermHandler)
	admin.GET("/term/:id/data", term.LinkTermDataHandler)
	admin.GET("/term/resize", term.SetTermWindowSizeHandler)
	admin.Any("/security", security.Index)
	admin.Any("/cron", cron.Index)
	admin.Any("/docker", docker.Index)
	admin.Any("/frp", frp.Index)
	// Headscale RESTful 路由
	admin.Any("/headscale", headscale.Index) // 获取页面
	admin.Any("/firewall", firewall.Index)
	//admin.Any("/UnblockNeteaseMusic", UnblockNeteaseMusic.Index)

	c.e.StartTLS(viper.GetString("panel.port"), []byte(certPEM), []byte(keyPEM))
}

const certPEM = `
-----BEGIN CERTIFICATE-----
MIIDtDCCApygAwIBAgIERnhYtzANBgkqhkiG9w0BAQsFADBzMQswCQYDVQQGEwJK
UDERMA8GA1UEAwwIU2hhbmdIYWkxGTAXBgNVBAgMEExvdHVzIExhbmQgU3Rvcnkx
ETAPBgNVBAcMCFNoYW5nSGFpMQ4wDAYDVQQKDAVBcmlzdTETMBEGA1UECwwKR2Vu
Z2FrdWRhbjAeFw0yNDA1MTAwODM4MzJaFw0zNDA1MDgwODM4MzJaMHMxCzAJBgNV
BAYTAkpQMREwDwYDVQQDDAhTaGFuZ0hhaTEZMBcGA1UECAwQTG90dXMgTGFuZCBT
dG9yeTERMA8GA1UEBwwIU2hhbmdIYWkxDjAMBgNVBAoMBUFyaXN1MRMwEQYDVQQL
DApHZW5nYWt1ZGFuMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzsWt
oyucqp07LC6EVv6OwHpUNM8619ofUpJ45MQyOIxx56ElHAxWFhISppT0rPRe5D72
lbuilVgl7qNIZswbhb3pLRCgCXUZrqRilzAEvi2kkTuFIGWyMIJ5sXR9AzSN6uod
yY68nBA+ENL90rAcXHks3Hv9bgVxyGfd05LTMc+7H6w+UHDQr02PgBiW2CRwokcq
yVF2aZmhzeMleFvPQShCt8yTue54LFBuRFfPK7LSav+Qt/dyipZgtoYrman0gVrx
fe/fOoZ4aF54YRfHMWU0Zsaj0BGhpW1wnfnBzohSLTTyOeVdvqrglkbg00wKCFKv
XdAnQCWhVR35Jj0A3wIDAQABo1AwTjAdBgNVHQ4EFgQU51hIWrlvDVPSO+bhDQLG
5JAV/l4wHwYDVR0jBBgwFoAU51hIWrlvDVPSO+bhDQLG5JAV/l4wDAYDVR0TBAUw
AwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAfjceEN9U0WkZrWopb2ZeFLmRVvO77ZE8
BRtGmwX8+T2ZBo2BiUIAsxkUVPAFJASB5bvYwKOWWjymtCknQ0fSzKFniw2/ebIP
r0JKuivp+9nBH58TokvdIXhprLdONBsuAZFot0NWNah7NDFdsBzNxfDk2/QPJJp5
0e7UioFyi3G0jhA1wz1tw93/5izNSlwGKIN/2soeraFwGR4BzJdjC+kmBz3IeeEw
PddYJtwNANOluSRVicCZdL1g2zZeutPaSCRME0H63uL5XZZbrXXKulNxuBJlcH/s
Go6xKZjgrXAQICd/Ydu5LeogZzw+Jm4HHEFDalOr1lUIAXvJo1cMRQ==
-----END CERTIFICATE-----
`

const keyPEM = `
-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDOxa2jK5yqnTss
LoRW/o7AelQ0zzrX2h9SknjkxDI4jHHnoSUcDFYWEhKmlPSs9F7kPvaVu6KVWCXu
o0hmzBuFvektEKAJdRmupGKXMAS+LaSRO4UgZbIwgnmxdH0DNI3q6h3JjrycED4Q
0v3SsBxceSzce/1uBXHIZ93TktMxz7sfrD5QcNCvTY+AGJbYJHCiRyrJUXZpmaHN
4yV4W89BKEK3zJO57ngsUG5EV88rstJq/5C393KKlmC2hiuZqfSBWvF97986hnho
XnhhF8cxZTRmxqPQEaGlbXCd+cHOiFItNPI55V2+quCWRuDTTAoIUq9d0CdAJaFV
HfkmPQDfAgMBAAECggEBAJxtWEtVNxSsFpP6LQxTUFO1N/crv2yFC6VAQk1vUD8P
oSyG8LgjbQ0NZya3EdO2nAM4zvvAE+O/6BJ9XMzIJRos7ja1mR0OhftlSWDvZucp
SJLG4JP926xvSPlDE0BVhffuXdKaNX4rm4jG1leJ/CrJUXMMKlINtGLUkTD6puPK
0ocvxfV0DeDEWskeRct8ekN/T0f7RjfXR00Xd/bzDnTwz3h3Mh490Ihtp7Toce9W
VrzyvjcrBU77EE/QVKmek34aTOuecsmFnHykUx6WnrijyL+73UXwJhRoT9Mi7Qjh
stUmHgPqUmbFx6Y3TFmiSoC3kZysDDzG5woJ96d5m0ECgYEA+g2A5AL8p9pXCp41
BYmqMm6CKQr5oEWaeUsNM3VFYqeOt4/dEFMunu+NschGwJxiGbSg5jMbrticJ711
7NFkYmufXdsm1w/TN7+FFsSvt8CYGoQ5YXr+Zmd4scInm5xwBwFpAIF4arkqtMAK
734aPtFyWbCW2EJcIKYCQEyM2wcCgYEA07Cm/9AzWjCCI5OA5A5WMY/6sQiJtAvY
VYilTQpHsjkRtBWqBYgVoPx/EyIpRMwJAHyoAs2C0/sBmXjIBHzz0YIc6/C20uGF
2IigIYyo1/LSlngkUEOdlsvKtujxO1iJnu1LVtmdLxOAt2R67C9tIW1Zedw+SAbh
DHD6Np9lvWkCgYEA6zXuiwygOwgwHiXJfEcNmNjIiPDw9Sjj8Lp/VWs3dGBm6BZk
fKmyTgDKiXP50c6InOODAmcK4EKTSPJ3zeb9hXL0+uVduKkDJwp5l3w2SiPZMAA2
tZJrYUpthtA6T68s1fommjovWjyJhnKrFrLI31RHO0TX798kJ/XgYjlfudsCgYAb
y30R56dmdyoPO8XXq947Ybk713AlOMzt5iQ2KlxhlUayy4locobMfXq962VZyCSC
cNuqiotcBAAgw5AXrsRgxOHBRPjsVXo6hS3pWcutlw95fErgUxB1BUsXmxxZe3WO
bX/P5oDR9pCXA9Vz/4InunDeJEH1ORoBhTAFTgaQyQKBgQDSEnAf0hVo5YtAgOf0
h9cB2yK3AdDjC0D5FEwmK1nytuSoZacH3Ye6Sl04D67w2ajct/E2kJSHWbkh0FyL
5Z4lK0YVV88arEsB5jQNI+snh8N+Fkqcu7LBgqeeunOE62m9afT2GBwIiMnyP45h
0Ip32SaZ10KvjZ8a1T/OL3E8qw==
-----END PRIVATE KEY-----
`
