//go:build linux || darwin || freebsd

package headscale

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *headscaleConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &headscaleConfig{
				ServerURL:         "https://myheadscale.example.com:443",
				ListenAddr:        "0.0.0.0:8080",
				MetricsListenAddr: "127.0.0.1:9090",
				GRPCListenAddr:    "127.0.0.1:50443",
				PrivateKeyPath:    "/var/lib/headscale/noise_private.key",
				IPv4Prefix:        "100.64.0.0/10",
				IPv6Prefix:        "fd7a:115c:a1e0::/48",
				BaseDomain:        "example.com",
			},
			wantErr: false,
		},
		{
			name: "invalid IPv4 prefix",
			config: &headscaleConfig{
				ServerURL:         "https://myheadscale.example.com:443",
				ListenAddr:        "0.0.0.0:8080",
				MetricsListenAddr: "127.0.0.1:9090",
				GRPCListenAddr:    "127.0.0.1:50443",
				PrivateKeyPath:    "/var/lib/headscale/noise_private.key",
				IPv4Prefix:        "invalid",
				IPv6Prefix:        "fd7a:115c:a1e0::/48",
				BaseDomain:        "example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid IPv6 prefix",
			config: &headscaleConfig{
				ServerURL:         "https://myheadscale.example.com:443",
				ListenAddr:        "0.0.0.0:8080",
				MetricsListenAddr: "127.0.0.1:9090",
				GRPCListenAddr:    "127.0.0.1:50443",
				PrivateKeyPath:    "/var/lib/headscale/noise_private.key",
				IPv4Prefix:        "100.64.0.0/10",
				IPv6Prefix:        "invalid",
				BaseDomain:        "example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid server URL",
			config: &headscaleConfig{
				ServerURL:         "://invalid-url",
				ListenAddr:        "0.0.0.0:8080",
				MetricsListenAddr: "127.0.0.1:9090",
				GRPCListenAddr:    "127.0.0.1:50443",
				PrivateKeyPath:    "/var/lib/headscale/noise_private.key",
				IPv4Prefix:        "100.64.0.0/10",
				IPv6Prefix:        "fd7a:115c:a1e0::/48",
				BaseDomain:        "example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := getDefaultConfig()
	
	if config == nil {
		t.Fatal("getDefaultConfig() returned nil")
	}
	
	// 验证默认值
	if config.ServerURL != "https://myheadscale.example.com:443" {
		t.Errorf("Expected default ServerURL to be 'https://myheadscale.example.com:443', got %s", config.ServerURL)
	}
	
	if config.ListenAddr != "0.0.0.0:8080" {
		t.Errorf("Expected default ListenAddr to be '0.0.0.0:8080', got %s", config.ListenAddr)
	}
	
	if config.IPv4Prefix != "100.64.0.0/10" {
		t.Errorf("Expected default IPv4Prefix to be '100.64.0.0/10', got %s", config.IPv4Prefix)
	}
	
	if config.IPv6Prefix != "fd7a:115c:a1e0::/48" {
		t.Errorf("Expected default IPv6Prefix to be 'fd7a:115c:a1e0::/48', got %s", config.IPv6Prefix)
	}
	
	if config.BaseDomain != "example.com" {
		t.Errorf("Expected default BaseDomain to be 'example.com', got %s", config.BaseDomain)
	}
}

func TestLoadServerConfig(t *testing.T) {
	config := &headscaleConfig{
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
	
	hsConfig, err := loadServerConfig(config)
	if err != nil {
		t.Fatalf("loadServerConfig() error = %v", err)
	}
	
	if hsConfig == nil {
		t.Fatal("loadServerConfig() returned nil config")
	}
	
	// 验证转换后的配置
	if hsConfig.ServerURL != config.ServerURL {
		t.Errorf("Expected ServerURL to be %s, got %s", config.ServerURL, hsConfig.ServerURL)
	}
	
	if hsConfig.Addr != config.ListenAddr {
		t.Errorf("Expected Addr to be %s, got %s", config.ListenAddr, hsConfig.Addr)
	}
	
	if hsConfig.MetricsAddr != config.MetricsListenAddr {
		t.Errorf("Expected MetricsAddr to be %s, got %s", config.MetricsListenAddr, hsConfig.MetricsAddr)
	}
	
	if hsConfig.GRPCAddr != config.GRPCListenAddr {
		t.Errorf("Expected GRPCAddr to be %s, got %s", config.GRPCListenAddr, hsConfig.GRPCAddr)
	}
	
	if hsConfig.BaseDomain != config.BaseDomain {
		t.Errorf("Expected BaseDomain to be %s, got %s", config.BaseDomain, hsConfig.BaseDomain)
	}
}