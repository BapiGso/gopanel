//go:build linux || darwin || freebsd

package headscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/juanfont/headscale/hscontrol"
	"github.com/juanfont/headscale/hscontrol/types"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	headscaleApp     *hscontrol.Headscale
	headscaleMutex   sync.RWMutex
	headscaleCancel  context.CancelFunc
	headscaleRunning bool
	configPath       = "/etc/headscale/config.json"
)

// HeadscaleManager 管理 Headscale 实例
type HeadscaleManager struct {
	app     *hscontrol.Headscale
	cancel  context.CancelFunc
	running bool
	mu      sync.RWMutex
}

var manager = &HeadscaleManager{}

// SaveConfig 保存配置到文件
func SaveConfig(config *headscaleConfig) error {
	// 确保配置目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 将配置序列化为 JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfig 从文件加载配置
func LoadConfig() (*headscaleConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果配置文件不存在，返回默认配置
			return getDefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config headscaleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// getDefaultConfig 返回默认配置
func getDefaultConfig() *headscaleConfig {
	return &headscaleConfig{
		ServerURL:         "https://myheadscale.example.com:443",
		ListenAddr:        "0.0.0.0:8080",
		MetricsListenAddr: "127.0.0.1:9090",
		GRPCListenAddr:    "127.0.0.1:50443",
		PrivateKeyPath:    "/var/lib/headscale/noise_private.key",
		IPv4Prefix:        "100.64.0.0/10",
		IPv6Prefix:        "fd7a:115c:a1e0::/48",
		BaseDomain:        "example.com",
		DatabaseType:      "sqlite",
		SqlitePath:        "/var/lib/headscale/db.sqlite",
	}
}

// StartHeadscale 启动 Headscale 服务
func StartHeadscale() error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.running {
		return errors.New("headscale is already running")
	}

	// 加载配置
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 转换为 Headscale 配置
	hsConfig, err := loadServerConfig(config)
	if err != nil {
		return fmt.Errorf("failed to convert config: %w", err)
	}

	// 创建 Headscale 实例
	app, err := hscontrol.NewHeadscale(hsConfig)
	if err != nil {
		return fmt.Errorf("failed to create headscale instance: %w", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 在后台启动 Headscale
	go func() {
		if err := app.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// 记录错误但不阻塞
			fmt.Printf("Headscale serve error: %v\n", err)
		}
	}()

	manager.app = app
	manager.cancel = cancel
	manager.running = true

	// 等待一小段时间确保服务启动
	time.Sleep(2 * time.Second)

	return nil
}

// StopHeadscale 停止 Headscale 服务
func StopHeadscale() error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if !manager.running {
		return errors.New("headscale is not running")
	}

	// 取消上下文
	if manager.cancel != nil {
		manager.cancel()
	}

	// 关闭 Headscale
	if manager.app != nil {
		// Headscale 没有提供优雅关闭的方法，这里只能取消上下文
		// 实际生产环境中可能需要更复杂的关闭逻辑
		manager.app = nil
	}

	manager.running = false
	return nil
}

// IsRunning 检查 Headscale 是否正在运行
func IsRunning() bool {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return manager.running
}

// RestartHeadscale 重启 Headscale 服务
func RestartHeadscale() error {
	if IsRunning() {
		if err := StopHeadscale(); err != nil {
			return fmt.Errorf("failed to stop headscale: %w", err)
		}
		// 等待服务完全停止
		time.Sleep(2 * time.Second)
	}

	if err := StartHeadscale(); err != nil {
		return fmt.Errorf("failed to start headscale: %w", err)
	}

	return nil
}
