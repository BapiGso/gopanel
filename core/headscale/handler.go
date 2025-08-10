//go:build linux || darwin || freebsd

package headscale

import (
	"fmt"
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
	ServerURL              string `form:"server_url" json:"server_url"`                             // 服务器URL，客户端将连接到的地址
	ListenAddr             string `form:"listen_addr" json:"listen_addr"`                           // 服务器监听地址
	MetricsListenAddr      string `form:"metrics_listen_addr" json:"metrics_listen_addr"`           // metrics监听地址
	GRPCListenAddr         string `form:"grpc_listen_addr" json:"grpc_listen_addr"`                 // gRPC监听地址
	PrivateKeyPath         string `form:"private_key_path" json:"private_key_path"`                 // Noise私钥路径
	IPv4Prefix             string `form:"ipv4_prefix" json:"ipv4_prefix"`                           // IPv4地址分配范围
	IPv6Prefix             string `form:"ipv6_prefix" json:"ipv6_prefix"`                           // IPv6地址分配范围
	BaseDomain             string `form:"base_domain" json:"base_domain"`                           // MagicDNS基础域名
	ACMEEmail              string `form:"acme_email" json:"acme_email"`                             // ACME注册邮箱
	TLSLetsEncryptHostname string `form:"tls_letsencrypt_hostname" json:"tls_letsencrypt_hostname"` // Let's Encrypt主机名
	TLSCertPath            string `form:"tls_cert_path" json:"tls_cert_path"`                       // TLS证书路径
	TLSKeyPath             string `form:"tls_key_path" json:"tls_key_path"`                         // TLS私钥路径
	UnixSocket             string `form:"unix_socket" json:"unix_socket"`                           // Unix套接字路径
	UnixSocketPermission   int    `form:"unix_socket_permission" json:"unix_socket_permission"`     // Unix套接字权限
	DatabaseType           string `form:"database_type" json:"database_type"`                       // 数据库类型
	SqlitePath             string `form:"sqlite_path" json:"sqlite_path"`                           // SQLite数据库路径
	DisableCheckUpdates    bool   `form:"disable_check_updates" json:"disable_check_updates"`       // 是否禁用更新检查
}

// ServiceAction 定义服务操作
type ServiceAction struct {
	Action string `json:"action"` // start, stop, restart
}

// Index 处理主页面请求 - GET /admin/headscale
func Index(c echo.Context) error {
	// 加载当前配置
	config, err := LoadConfig()
	if err != nil {
		// 如果加载失败，使用默认配置
		config = getDefaultConfig()
	}

	return c.Render(http.StatusOK, "headscale.template", map[string]any{
		"config":          config,
		"headscaleEnable": viper.GetBool("enable.headscale"),
	})
}

// GetConfig 获取配置 - GET /admin/headscale/config
func GetConfig(c echo.Context) error {
	config, err := LoadConfig()
	if err != nil {
		// 如果配置不存在，返回默认配置
		config = getDefaultConfig()
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateConfig 更新配置 - PUT /admin/headscale/config
func UpdateConfig(c echo.Context) error {
	req := &headscaleConfig{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// 验证配置
	if err := validateConfig(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": fmt.Sprintf("Invalid configuration: %v", err),
		})
	}

	// 保存配置
	if err := SaveConfig(req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to save configuration: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Configuration updated successfully",
		"config":  req,
	})
}

// GetServiceStatus 获取服务状态 - GET /admin/headscale/service
func GetServiceStatus(c echo.Context) error {
	status := "stopped"
	if IsRunning() {
		status = "running"
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": status,
		"uptime": getUptime(), // 可以添加运行时间等额外信息
	})
}

// UpdateService 更新服务状态 - PUT /admin/headscale/service
func UpdateService(c echo.Context) error {
	var action ServiceAction
	if err := c.Bind(&action); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	var err error
	var message string

	switch action.Action {
	case "start":
		err = StartHeadscale()
		message = "Headscale started successfully"
	case "stop":
		err = StopHeadscale()
		message = "Headscale stopped successfully"
	case "restart":
		err = RestartHeadscale()
		message = "Headscale restarted successfully"
	default:
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": fmt.Sprintf("Invalid action: %s. Valid actions are: start, stop, restart", action.Action),
		})
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to %s Headscale: %v", action.Action, err),
		})
	}

	// 返回新状态
	status := "stopped"
	if IsRunning() {
		status = "running"
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": message,
		"status":  status,
	})
}

// getUptime 获取服务运行时间
func getUptime() string {
	// 这里可以实现获取服务运行时间的逻辑
	// 暂时返回空字符串
	return ""
}

// validateConfig 验证配置的有效性
func validateConfig(c *headscaleConfig) error {
	// 验证 IPv4 前缀
	if _, err := netip.ParsePrefix(c.IPv4Prefix); err != nil {
		return fmt.Errorf("invalid IPv4 prefix: %w", err)
	}

	// 验证 IPv6 前缀
	if _, err := netip.ParsePrefix(c.IPv6Prefix); err != nil {
		return fmt.Errorf("invalid IPv6 prefix: %w", err)
	}

	// 验证服务器 URL
	if _, err := url.Parse(c.ServerURL); err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// 设置默认值
	if c.DatabaseType == "" {
		c.DatabaseType = "sqlite"
	}
	if c.SqlitePath == "" {
		c.SqlitePath = "/var/lib/headscale/db.sqlite"
	}
	if c.UnixSocket == "" {
		c.UnixSocket = "/var/run/headscale.sock"
	}
	if c.UnixSocketPermission == 0 {
		c.UnixSocketPermission = 0600
	}

	return nil
}

// loadServerConfig 将配置转换为 Headscale 的配置格式
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
		Log: types.LogConfig{
			Format: "text",
			Level:  "info",
		},
		DisableUpdateCheck: c.DisableCheckUpdates, // 禁用更新检查
		Database: types.DatabaseConfig{
			Type:  c.DatabaseType, // 数据库类型
			Debug: false,          // 是否启用数据库调试
			Sqlite: types.SqliteConfig{
				Path:          c.SqlitePath, // SQLite数据库路径
				WriteAheadLog: true,         // 启用预写日志以提高性能
			},
		},
		DERP: types.DERPConfig{
			ServerEnabled:                      false,               // 是否启用DERP服务器
			AutomaticallyAddEmbeddedDerpRegion: false,               // 是否自动添加嵌入式DERP区域
			URLs:                               []url.URL{*derpURL}, // DERP服务器URL列表
			AutoUpdate:                         true,                // 是否自动更新DERP地图
			UpdateFrequency:                    24 * time.Hour,      // DERP地图更新频率
		},
		TLS: types.TLSConfig{
			LetsEncrypt: types.LetsEncryptConfig{
				Hostname:  c.TLSLetsEncryptHostname,
				Listen:    ":http",
				CacheDir:  "/var/lib/headscale/cache",
				Challenge: "HTTP-01",
			},
			CertPath: c.TLSCertPath,
			KeyPath:  c.TLSKeyPath,
		},
		ACMEURL:   "https://acme-v02.api.letsencrypt.org/directory", // ACME服务器URL
		ACMEEmail: c.ACMEEmail,                                      // ACME注册邮箱
		DNSConfig: types.DNSConfig{
			MagicDNS:   true,
			BaseDomain: c.BaseDomain,
			Nameservers: types.Nameservers{
				Global: []string{
					"1.1.1.1",
					"1.0.0.1",
					"2606:4700:4700::1111",
					"2606:4700:4700::1001",
				},
			},
		},
		UnixSocket:           c.UnixSocket,           // Unix套接字路径
		UnixSocketPermission: c.UnixSocketPermission, // Unix套接字权限
	}, nil
}
