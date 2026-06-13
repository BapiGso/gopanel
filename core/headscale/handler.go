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
	hsCancel  context.CancelFunc    // 用于停止服务的开关
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
	// 获取请求动作类型
	action := c.QueryParam("status")
	switch c.Request().Method {

	// 1. 处理 POST 请求 (控制指令：启动、停止、检查状态)
	case "POST":
		hsMutex.Lock()
		defer hsMutex.Unlock()

		switch action {
		case "start":
			// 如果已经在运行，直接返回成功
			if hsRunning {
				return c.JSON(200, map[string]any{"success": true, "message": "Already running"})
			}

			// 绑定参数
			req := &headscaleConfig{}
			if err := c.Bind(req); err != nil {
				return c.JSON(400, map[string]any{"success": false, "message": "Invalid config"})
			}

			// 转换配置
			cfg, err := loadServerConfig(req)
			if err != nil {
				return c.JSON(400, map[string]any{"success": false, "message": err.Error()})
			}

			// 初始化 Headscale
			app, err := hscontrol.NewHeadscale(cfg)
			if err != nil {
				return c.JSON(500, map[string]any{"success": false, "message": "Init failed: " + err.Error()})
			}

			var standaloneSTUN *standaloneSTUNServer
			if req.shouldStartStandaloneSTUN() {
				standaloneSTUN, err = startStandaloneSTUN(req.stunListenAddr())
				if err != nil {
					return c.JSON(500, map[string]any{"success": false, "message": "STUN start failed: " + err.Error()})
				}
			}

			// 创建用于停止的 Context
			ctx, cancel := context.WithCancel(context.Background())

			// 更新全局状态
			hsApp = app
			hsCancel = cancel
			hsRunning = true
			hsError = ""
			stunSrv = standaloneSTUN

			// 【关键点】放入 goroutine 异步运行，防止阻塞 HTTP 请求
			go func() {
				// 注意：这里需要 Headscale 支持 Context 传入或者只是单纯运行
				// 目前 Headscale 的 Serve 通常是阻塞的
				// 如果需要支持优雅关闭，可能需要修改 Headscale 源码或使用其提供的 Shutdown 方法
				// 这里简单处理：启动服务
				if err := app.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					hsMutex.Lock()
					hsRunning = false
					hsError = err.Error()
					stopStandaloneSTUNLocked()
					hsMutex.Unlock()
					fmt.Println("Headscale stopped unexpectedly:", err)
				} else {
					// 正常退出
					hsMutex.Lock()
					hsRunning = false
					stopStandaloneSTUNLocked()
					hsMutex.Unlock()
				}
			}()

			// 这里稍微利用 Context 做一个假装的生命周期管理（Headscale原生Serve如果不接受Context，强制停止比较麻烦）
			// 为了演示，我们保存了 cancel，实际停止时可能需要关闭 listener
			go func() {
				<-ctx.Done()
				// 当 cancel 被调用时，尝试关闭 app (如果 app 暴露了 Shutdown)
				// hsApp.Shutdown()
			}()

			return c.JSON(200, map[string]any{"success": true, "message": "Headscale started in background"})

		case "stop":
			if !hsRunning {
				return c.JSON(200, map[string]any{"success": false, "message": "Not running"})
			}

			// 触发停止
			if hsCancel != nil {
				hsCancel() // 触发 Context 取消
			}

			// 注意：由于 Go 的 http.Server 需要显式 Shutdown，
			// 如果 headscale 库没暴露 Shutdown 方法，这里可能关不掉端口。
			// 假设 hscontrol 内部处理了关闭逻辑，或者你可能需要重启整个主进程。

			hsRunning = false
			hsApp = nil
			stopStandaloneSTUNLocked()
			return c.JSON(200, map[string]any{"success": true, "message": "Stop signal sent"})

		case "check": // 前端轮询用
			statusStr := "stopped"
			if hsRunning {
				statusStr = "running"
			}
			return c.JSON(200, map[string]any{
				"status": statusStr,
				"error":  hsError,
			})

		default:
			// 可以在这里处理保存配置逻辑
			// saveConfigToDisk(c)
			return c.JSON(200, map[string]any{"success": true, "message": "Config saved (mock)"})
		}

	case "GET":
		// 可以在这里读取配置文件回显
		return c.Render(http.StatusOK, "headscale.template", map[string]any{
			"headscaleEnable": hsRunning,
		})
	}

	return echo.ErrMethodNotAllowed
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
		ServerRegionID:                     derpRegionID,
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
		ServerURL:                      c.ServerURL,
		Addr:                           c.ListenAddr,
		MetricsAddr:                    c.MetricsListenAddr,
		GRPCAddr:                       "127.0.0.1:50443",
		GRPCAllowInsecure:              true,
		EphemeralNodeInactivityTimeout: 30 * time.Minute,
		PrefixV4:                       &prefix4,
		PrefixV6:                       &prefix6,
		IPAllocation:                   types.IPAllocationStrategySequential,
		NoisePrivateKeyPath:            c.PrivateKeyPath,
		BaseDomain:                     c.BaseDomain,
		Log:                            types.LogConfig{Level: 1},
		DisableUpdateCheck:             true,
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

	return &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{
			regionID: {
				RegionID:   regionID,
				RegionCode: regionCode,
				RegionName: regionName,
				Nodes: []*tailcfg.DERPNode{
					{
						Name:     fmt.Sprintf("%dstun", regionID),
						RegionID: regionID,
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
