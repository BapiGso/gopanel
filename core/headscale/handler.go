//go:build linux || darwin || freebsd

package headscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/juanfont/headscale/hscontrol"
	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/labstack/echo/v5"
	"tailscale.com/net/stun"
	"tailscale.com/tailcfg"
)

// ================= 全局变量区 =================
// 必须放在函数外面，否则每次请求进来都是新的，无法管理状态
var (
	hsApp     *hscontrol.Headscale  // Headscale 实例
	hsRunning bool                  // 运行状态标记
	hsMutex   sync.Mutex            // 线程锁，防止同时点击启动炸掉
	hsError   string                // 记录启动时的报错信息
	stunSrv   *standaloneSTUNServer // 独立 STUN 服务
)

const (
	defaultDERPRegionID   = 999
	defaultDERPRegionCode = "gopanel"
	defaultDERPRegionName = "GoPanel Embedded DERP"
	defaultSTUNListenAddr = "0.0.0.0:3478"
)

// headscaleConfig 前端表单结构
type headscaleConfig struct {
	ServerURL         string `form:"server_url"`
	ListenAddr        string `form:"listen_addr"`
	MetricsListenAddr string `form:"metrics_listen_addr"`
	PrivateKeyPath    string `form:"private_key_path"`
	IPv4Prefix        string `form:"ipv4_prefix"`
	IPv6Prefix        string `form:"ipv6_prefix"`
	BaseDomain        string `form:"base_domain"`
	STUNEnabled       bool   `form:"stun_enabled"`
	STUNListenAddr    string `form:"stun_listen_addr" default:"0.0.0.0:3478"`
	DERPEnabled       bool   `form:"derp_enabled"`
	DERPRegionID      int    `form:"derp_region_id" default:"999"`
	DERPRegionCode    string `form:"derp_region_code" default:"gopanel"`
	DERPRegionName    string `form:"derp_region_name" default:"GoPanel Embedded DERP"`
	DERPSTUNAddr      string `form:"derp_stun_addr" default:"0.0.0.0:3478"` // 兼容旧表单字段
	DERPVerifyClient  string `form:"derp_verify_clients"`                   // "on" or ""
}

// Index 是唯一的入口函数
func Index(c *echo.Context) error {
	hsMutex.Lock()
	running := hsRunning
	hsMutex.Unlock()
	return c.Render(http.StatusOK, "headscale.template", map[string]any{"headscaleEnable": running})
}

// 辅助函数：配置转换
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
	stunAddr := c.stunListenAddr()
	derpRegionID := c.derpRegionID()
	derpRegionCode := c.derpRegionCode()
	derpRegionName := c.derpRegionName()

	// 获取当前路径作为数据目录
	cwd, _ := os.Getwd()
	dataDir := cwd + "/data/headscale"
	_ = os.MkdirAll(dataDir, 0755)

	derpConfig := types.DERPConfig{
		ServerEnabled:                      c.DERPEnabled,
		AutomaticallyAddEmbeddedDerpRegion: c.DERPEnabled,
		ServerRegionID:                     tailcfg.DERPRegionID(derpRegionID),
		ServerRegionCode:                   derpRegionCode,
		ServerRegionName:                   derpRegionName,
		STUNAddr:                           stunAddr,
		ServerPrivateKeyPath:               dataDir + "/derp_server.key",
		ServerVerifyClients:                c.DERPVerifyClient == "on",
		URLs:                               []url.URL{*derpURL},
		AutoUpdate:                         true,
		UpdateFrequency:                    24 * time.Hour,
	}

	if c.STUNEnabled && !c.DERPEnabled {
		stunOnlyMap, err := c.stunOnlyDERPMap(stunAddr, derpRegionID, derpRegionCode, derpRegionName)
		if err != nil {
			return nil, err
		}
		derpConfig.DERPMap = stunOnlyMap
	}

	return &types.Config{
		ServerURL:         c.ServerURL,
		Addr:              c.ListenAddr,
		MetricsAddr:       c.MetricsListenAddr,
		GRPCAddr:          "127.0.0.1:50443",
		GRPCAllowInsecure: true,
		Node: types.NodeConfig{
			Ephemeral: types.EphemeralConfig{
				InactivityTimeout: 30 * time.Minute,
			},
		},
		PrefixV4:            &prefix4,
		PrefixV6:            &prefix6,
		IPAllocation:        types.IPAllocationStrategySequential,
		NoisePrivateKeyPath: c.PrivateKeyPath,
		BaseDomain:          c.BaseDomain,
		Log:                 types.LogConfig{Level: 1},
		DisableUpdateCheck:  true,
		Database: types.DatabaseConfig{
			Type:  "sqlite3",
			Debug: false,
			Sqlite: types.SqliteConfig{
				Path:          dataDir + "/headscale.db",
				WriteAheadLog: true,
			},
		},
		DERP:                 derpConfig,
		TLS:                  types.TLSConfig{},
		DNSConfig:            types.DNSConfig{},
		UnixSocket:           dataDir + "/headscale.sock",
		UnixSocketPermission: 0755,
		Tuning: types.Tuning{
			NotifierSendTimeout:            800 * time.Millisecond,
			BatchChangeDelay:               800 * time.Millisecond,
			NodeMapSessionBufferedChanSize: 30,
			BatcherWorkers:                 runtime.NumCPU(),
			NodeStoreBatchSize:             100,
			NodeStoreBatchTimeout:          500 * time.Millisecond,
		},
	}, nil
}

func (c *headscaleConfig) stunListenAddr() string {
	if addr := strings.TrimSpace(c.STUNListenAddr); addr != "" {
		return addr
	}
	if addr := strings.TrimSpace(c.DERPSTUNAddr); addr != "" {
		return addr
	}
	return defaultSTUNListenAddr
}

func (c *headscaleConfig) derpRegionID() int {
	if c.DERPRegionID != 0 {
		return c.DERPRegionID
	}
	return defaultDERPRegionID
}

func (c *headscaleConfig) derpRegionCode() string {
	if code := strings.TrimSpace(c.DERPRegionCode); code != "" {
		return code
	}
	return defaultDERPRegionCode
}

func (c *headscaleConfig) derpRegionName() string {
	if name := strings.TrimSpace(c.DERPRegionName); name != "" {
		return name
	}
	return defaultDERPRegionName
}

func (c *headscaleConfig) shouldStartStandaloneSTUN() bool {
	return c.STUNEnabled && !c.DERPEnabled
}

func (c *headscaleConfig) stunOnlyDERPMap(stunAddr string, regionID int, regionCode, regionName string) (*tailcfg.DERPMap, error) {
	serverURL, err := url.Parse(c.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("parsing server URL for STUN map: %w", err)
	}

	host := serverURL.Hostname()
	if host == "" {
		return nil, fmt.Errorf("server URL host is required for STUN map")
	}

	_, portStr, err := net.SplitHostPort(stunAddr)
	if err != nil {
		return nil, fmt.Errorf("parsing STUN listen address: %w", err)
	}

	stunPort, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("parsing STUN port: %w", err)
	}
	if stunPort <= 0 || stunPort > 65535 {
		return nil, fmt.Errorf("STUN port out of range: %d", stunPort)
	}

	derpRegionID := tailcfg.DERPRegionID(regionID)

	return &tailcfg.DERPMap{
		Regions: map[tailcfg.DERPRegionID]*tailcfg.DERPRegion{
			derpRegionID: {
				RegionID:   derpRegionID,
				RegionCode: regionCode,
				RegionName: regionName,
				Nodes: []*tailcfg.DERPNode{
					{
						Name:     fmt.Sprintf("%dstun", regionID),
						RegionID: derpRegionID,
						HostName: host,
						STUNPort: stunPort,
						STUNOnly: true,
					},
				},
			},
		},
	}, nil
}

type standaloneSTUNServer struct {
	conn   *net.UDPConn
	cancel context.CancelFunc
	done   chan struct{}
}

func startStandaloneSTUN(addr string) (*standaloneSTUNServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolving STUN address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("opening STUN listener: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := &standaloneSTUNServer{
		conn:   conn,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go server.serve(ctx)

	return server, nil
}

func stopStandaloneSTUNLocked() {
	if stunSrv == nil {
		return
	}

	stunSrv.stop()
	stunSrv = nil
}

func (s *standaloneSTUNServer) stop() {
	if s == nil {
		return
	}

	s.cancel()
	_ = s.conn.Close()

	select {
	case <-s.done:
	case <-time.After(time.Second):
	}
}

func (s *standaloneSTUNServer) serve(ctx context.Context) {
	defer close(s.done)

	var buf [64 << 10]byte
	for {
		bytesRead, udpAddr, err := s.conn.ReadFromUDP(buf[:])
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}

			time.Sleep(time.Second)
			continue
		}

		packet := buf[:bytesRead]
		if !stun.Is(packet) {
			continue
		}

		txid, err := stun.ParseBindingRequest(packet)
		if err != nil {
			continue
		}

		addr, ok := netip.AddrFromSlice(udpAddr.IP)
		if !ok {
			continue
		}

		response := stun.Response(txid, netip.AddrPortFrom(addr, uint16(udpAddr.Port)))
		_, _ = s.conn.WriteToUDP(response, udpAddr)
	}
}
