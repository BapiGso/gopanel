//go:build linux || darwin || freebsd

package headscale

import (
	"errors"
	"fmt"
	"github.com/juanfont/headscale/hscontrol"
	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/juanfont/headscale/hscontrol/util"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go4.org/netipx"
	"io"
	"net/http"
	"net/netip"
	"os"
	"tailscale.com/net/tsaddr"
	"time"
)

// headscaleConfig 定义了Headscale的配置结构
type headscaleConfig struct {
	ServerURL         string `form:"server_url"`          // 服务器URL，客户端将连接到的地址
	ListenAddr        string `form:"listen_addr"`         // 服务器监听地址
	MetricsListenAddr string `form:"metrics_listen_addr"` // metrics监听地址
	GRPCListenAddr    string `form:"grpc_listen_addr"`    // gRPC监听地址
	PrivateKeyPath    string `form:"private_key_path"`    // Noise私钥路径
	IPv4Prefix        string `form:"ipv4_prefix"`         // IPv4地址分配范围
	IPv6Prefix        string `form:"ipv6_prefix"`         // IPv6地址分配范围
	BaseDomain        string `form:"base_domain"`         // MagicDNS基础域名
}

const confPath = "/etc/headscale/config.yaml"

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
			//cfg, err := types.LoadServerConfig()
			//if err != nil {
			//	return err
			//}
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
	case "PUT":
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(confPath, data, 0644); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "GET":
		file, err := os.ReadFile(confPath)
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "headscale.template", map[string]any{
			"headscaleConfig": string(file),
			"headscaleEnable": viper.Get("enable.frps").(bool),
		})
	}

	return echo.ErrMethodNotAllowed
}

func loadServerConfig(c *headscaleConfig) (*types.Config, error) {
	prefix4, err := prefixV4()
	if err != nil {
		return nil, err
	}

	prefix6, err := prefixV6()
	if err != nil {
		return nil, err
	}
	allocStr := "sequential"
	var alloc types.IPAllocationStrategy
	switch allocStr {
	case string(types.IPAllocationStrategySequential):
		alloc = types.IPAllocationStrategySequential
	case string(types.IPAllocationStrategyRandom):
		alloc = types.IPAllocationStrategyRandom
	default:
		return nil, fmt.Errorf("config error, prefixes.allocation is set to %s, which is not a valid strategy, allowed options: %s, %s", allocStr, types.IPAllocationStrategySequential, types.IPAllocationStrategyRandom)
	}
	return &types.Config{
		ServerURL:          c.ServerURL,
		Addr:               c.ListenAddr,
		MetricsAddr:        c.MetricsListenAddr,
		GRPCAddr:           c.GRPCListenAddr,
		GRPCAllowInsecure:  false,
		DisableUpdateCheck: false,

		PrefixV4:     prefix4,
		PrefixV6:     prefix6,
		IPAllocation: alloc,

		NoisePrivateKeyPath: util.AbsolutePathFromConfigPath(
			c.PrivateKeyPath,
		),
		//BaseDomain: dnsConfig.BaseDomain,
		//
		//DERP: derpConfig,

		EphemeralNodeInactivityTimeout: viper.GetDuration(
			"ephemeral_node_inactivity_timeout",
		),

		Database: types.DatabaseConfig{
			Type:  "sqlite3",
			Debug: false,
			Sqlite: types.SqliteConfig{
				Path:          "/var/lib/headscale/db.sqlite",
				WriteAheadLog: false,
			},
		},
		//
		//TLS: tlsConfig(),
		//
		//DNSConfig:             dnsToTailcfgDNS(dnsConfig),
		//DNSUserNameInMagicDNS: dnsConfig.UserNameInMagicDNS,

		ACMEEmail: "",
		ACMEURL:   "https://acme-v02.api.letsencrypt.org/directory",

		UnixSocket:           viper.GetString("unix_socket"),
		UnixSocketPermission: util.GetFileMode("unix_socket_permission"),

		//OIDC: OIDCConfig{
		//	OnlyStartIfOIDCIsAvailable: viper.GetBool(
		//		"oidc.only_start_if_oidc_is_available",
		//	),
		//	Issuer:           viper.GetString("oidc.issuer"),
		//	ClientID:         viper.GetString("oidc.client_id"),
		//	ClientSecret:     oidcClientSecret,
		//	Scope:            viper.GetStringSlice("oidc.scope"),
		//	ExtraParams:      viper.GetStringMapString("oidc.extra_params"),
		//	AllowedDomains:   viper.GetStringSlice("oidc.allowed_domains"),
		//	AllowedUsers:     viper.GetStringSlice("oidc.allowed_users"),
		//	AllowedGroups:    viper.GetStringSlice("oidc.allowed_groups"),
		//	StripEmaildomain: viper.GetBool("oidc.strip_email_domain"),
		//	Expiry: func() time.Duration {
		//		// if set to 0, we assume no expiry
		//		if value := viper.GetString("oidc.expiry"); value == "0" {
		//			return maxDuration
		//		} else {
		//			expiry, err := model.ParseDuration(value)
		//			if err != nil {
		//				log.Warn().Msg("failed to parse oidc.expiry, defaulting back to 180 days")
		//
		//				return defaultOIDCExpiryTime
		//			}
		//
		//			return time.Duration(expiry)
		//		}
		//	}(),
		//	UseExpiryFromToken: viper.GetBool("oidc.use_expiry_from_token"),
		//},

		//LogTail:             logTailConfig,
		RandomizeClientPort: false,
		//
		//Policy: policyConfig(),
		//
		//CLI: CLIConfig{
		//	Address:  viper.GetString("cli.address"),
		//	APIKey:   viper.GetString("cli.api_key"),
		//	Timeout:  viper.GetDuration("cli.timeout"),
		//	Insecure: viper.GetBool("cli.insecure"),
		//},
		//
		//Log: logConfig,

		// TODO(kradalby): Document these settings when more stable
		Tuning: types.Tuning{
			NotifierSendTimeout:            time.Duration(1 * time.Minute),
			BatchChangeDelay:               time.Duration(1 * time.Second),
			NodeMapSessionBufferedChanSize: 256,
		},
	}, nil
}

func prefixV4() (*netip.Prefix, error) {
	prefixV4Str := viper.GetString("prefixes.v4")

	if prefixV4Str == "" {
		return nil, nil
	}

	prefixV4, err := netip.ParsePrefix(prefixV4Str)
	if err != nil {
		return nil, fmt.Errorf("parsing IPv4 prefix from config: %w", err)
	}

	builder := netipx.IPSetBuilder{}
	builder.AddPrefix(tsaddr.CGNATRange())
	ipSet, _ := builder.IPSet()
	if !ipSet.ContainsPrefix(prefixV4) {
		log.Warn().
			Msgf("Prefix %s is not in the %s range. This is an unsupported configuration.",
				prefixV4Str, tsaddr.CGNATRange())
	}

	return &prefixV4, nil
}

func prefixV6() (*netip.Prefix, error) {
	prefixV6Str := viper.GetString("prefixes.v6")

	if prefixV6Str == "" {
		return nil, nil
	}

	prefixV6, err := netip.ParsePrefix(prefixV6Str)
	if err != nil {
		return nil, fmt.Errorf("parsing IPv6 prefix from config: %w", err)
	}

	builder := netipx.IPSetBuilder{}
	builder.AddPrefix(tsaddr.TailscaleULARange())
	ipSet, _ := builder.IPSet()

	if !ipSet.ContainsPrefix(prefixV6) {
		log.Warn().
			Msgf("Prefix %s is not in the %s range. This is an unsupported configuration.",
				prefixV6Str, tsaddr.TailscaleULARange())
	}

	return &prefixV6, nil
}
