//go:build linux || darwin || freebsd

package headscale

import (
	"errors"
	"fmt"
	"github.com/juanfont/headscale/hscontrol"
	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

// headscaleConfig 定义了Headscale的配置结构
type headscaleConfig struct {
	ServerURL              string `form:"server_url"`               // 服务器URL，客户端将连接到的地址
	ListenAddr             string `form:"listen_addr"`              // 服务器监听地址
	MetricsListenAddr      string `form:"metrics_listen_addr"`      // metrics监听地址
	GRPCListenAddr         string `form:"grpc_listen_addr"`         // gRPC监听地址
	PrivateKeyPath         string `form:"private_key_path"`         // Noise私钥路径
	IPv4Prefix             string `form:"ipv4_prefix"`              // IPv4地址分配范围
	IPv6Prefix             string `form:"ipv6_prefix"`              // IPv6地址分配范围
	BaseDomain             string `form:"base_domain"`              // MagicDNS基础域名
	ACMEEmail              string `form:"acme_email"`               // ACME注册邮箱
	TLSLetsEncryptHostname string `form:"tls_letsencrypt_hostname"` // Let's Encrypt主机名
	TLSCertPath            string `form:"tls_cert_path"`            // TLS证书路径
	TLSKeyPath             string `form:"tls_key_path"`             // TLS私钥路径
	UnixSocket             string `form:"unix_socket"`              // Unix套接字路径
	UnixSocketPermission   int    `form:"unix_socket_permission"`   // Unix套接字权限
	DatabaseType           string `form:"database_type"`            // 数据库类型
	SqlitePath             string `form:"sqlite_path"`              // SQLite数据库路径
	DisableCheckUpdates    bool   `form:"disable_check_updates"`    // 是否禁用更新检查
}

//const confPath = "/etc/headscale/config.yaml"

func Index(c echo.Context) error {
	req := &headscaleConfig{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	switch c.Request().Method {
	case "POST":
		if c.QueryParam("status") == "start" {
			cfg, err := loadServerConfig(req)
			if err != nil {
				return err
			}
			app, err := hscontrol.NewHeadscale(cfg)
			if err != nil {
				return err
			}
			if err = app.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}

		return c.JSON(200, "success")
	//case "PUT":
	//	data, err := io.ReadAll(c.Request().Body)
	//	defer c.Request().Body.Close()
	//	if err != nil {
	//		return err
	//	}
	//	if err := os.WriteFile(confPath, data, 0644); err != nil {
	//		return err
	//	}
	//	return c.JSON(200, "success")
	case "GET":
		//file, err := os.ReadFile(confPath)
		//if err != nil {
		//	return err
		//}
		return c.Render(http.StatusOK, "headscale.template", map[string]any{
			//"headscaleConfig": string(file),
			"headscaleEnable": viper.Get("enable.frps").(bool),
		})
	}

	return echo.ErrMethodNotAllowed
}

func loadServerConfig(c *headscaleConfig) (*types.Config, error) {
	prefix4, err := netip.ParsePrefix(c.IPv4Prefix)
	if err != nil {
		return nil, fmt.Errorf("parsing IPv4 prefix: %w", err)
	}

	prefix6, err := netip.ParsePrefix(c.IPv6Prefix)
	if err != nil {
		return nil, fmt.Errorf("parsing IPv6 prefix: %w", err)
	}

	derpURL, _ := url.Parse("https://controlplane.tailscale.com/derpmap/default")

	return &types.Config{
		ServerURL:                      c.ServerURL,                          // 客户端将连接到的服务器URL
		Addr:                           c.ListenAddr,                         // 服务器监听地址
		MetricsAddr:                    c.MetricsListenAddr,                  // metrics监听地址
		GRPCAddr:                       c.GRPCListenAddr,                     // gRPC监听地址
		GRPCAllowInsecure:              false,                                // 是否允许不安全的gRPC连接
		EphemeralNodeInactivityTimeout: 30 * time.Minute,                     // 临时节点不活动超时时间
		PrefixV4:                       &prefix4,                             // IPv4地址分配范围
		PrefixV6:                       &prefix6,                             // IPv6地址分配范围
		IPAllocation:                   types.IPAllocationStrategySequential, // IP分配策略
		NoisePrivateKeyPath:            c.PrivateKeyPath,                     // Noise协议私钥路径
		BaseDomain:                     c.BaseDomain,                         // MagicDNS基础域名
		Log:                            types.LogConfig{},                    // 日志配置
		DisableUpdateCheck:             true,                                 // 禁用更新检查
		Database: types.DatabaseConfig{
			Type:  "sqlite", // 数据库类型
			Debug: false,    // 是否启用数据库调试
			Sqlite: types.SqliteConfig{
				Path:          "/var/lib/headscale/db.sqlite", // SQLite数据库路径
				WriteAheadLog: false,                          // 是否启用预写日志
			},
		},
		DERP: types.DERPConfig{
			ServerEnabled:                      false,               // 是否启用DERP服务器
			AutomaticallyAddEmbeddedDerpRegion: false,               // 是否自动添加嵌入式DERP区域
			URLs:                               []url.URL{*derpURL}, // DERP服务器URL列表
			AutoUpdate:                         true,                // 是否自动更新DERP地图
			UpdateFrequency:                    24 * time.Hour,      // DERP地图更新频率
		},

		TLS:       types.TLSConfig{},                                // TLS配置
		ACMEURL:   "https://acme-v02.api.letsencrypt.org/directory", // ACME服务器URL
		ACMEEmail: "",                                               // ACME注册邮箱
		DNSConfig: types.DNSConfig{},                                // DNS配置
		//DNSUserNameInMagicDNS: false,                                            // 是否在MagicDNS中使用用户名
		UnixSocket:           "/var/run/headscale.sock", // Unix套接字路径
		UnixSocketPermission: 0600,                      // Unix套接字权限
	}, nil
}
