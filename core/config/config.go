package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

const filePath = "gopanel_config.json"

type Config struct {
	Panel  PanelConfig  `json:"panel"`
	WebDAV WebDAVConfig `json:"webdav"`
	Enable EnableConfig `json:"enable"`
}

type PanelConfig struct {
	Port     string `json:"port"`
	Path     string `json:"path"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type WebDAVConfig struct {
	Enable   bool   `json:"enable"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type EnableConfig struct {
	Caddy     bool `json:"caddy"`
	Frps      bool `json:"frps"`
	Frpc      bool `json:"frpc"`
	Headscale bool `json:"headscale"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	cfg  Config
}

var global = &Store{path: filePath}

func init() {
	if err := global.load(); err != nil {
		fmt.Printf("read config: %v\n", err)
		return
	}
	printStartupInfo(global.snapshot())
}

func Init() error {
	return global.load()
}

func Snapshot() Config {
	return global.snapshot()
}

func Get(path string) any {
	global.mu.RLock()
	defer global.mu.RUnlock()
	return get(global.cfg, path)
}

func String(path string) string {
	value, _ := Get(path).(string)
	return value
}

func Bool(path string) bool {
	value, _ := Get(path).(bool)
	return value
}

func Write(path string, value any) error {
	return global.write(path, value)
}

func Update(fn func(*Config)) error {
	return global.update(fn)
}

func (s *Store) snapshot() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		if err := writeConfigFile(s.path, Default()); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	s.cfg = cfg
	return nil
}

func (s *Store) write(path string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.cfg
	if err := set(&next, path, value); err != nil {
		return err
	}
	if err := writeConfigFile(s.path, next); err != nil {
		return err
	}

	s.cfg = next
	return nil
}

func (s *Store) update(fn func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.cfg
	fn(&next)
	if err := writeConfigFile(s.path, next); err != nil {
		return err
	}

	s.cfg = next
	return nil
}

func Default() Config {
	return Config{
		Panel: PanelConfig{
			Port:     ":8443",
			Path:     generateRandomString(4),
			Username: generateRandomString(6),
			Password: generateRandomString(6),
		},
		WebDAV: WebDAVConfig{
			Enable:   false,
			Username: generateRandomString(3),
			Password: generateRandomString(3),
		},
		Enable: EnableConfig{
			Caddy:     false,
			Frps:      false,
			Frpc:      false,
			Headscale: false,
		},
	}
}

func writeConfigFile(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func generateRandomString(n int) string {
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func get(cfg Config, path string) any {
	switch path {
	case "panel.port":
		return cfg.Panel.Port
	case "panel.path":
		return cfg.Panel.Path
	case "panel.username":
		return cfg.Panel.Username
	case "panel.password":
		return cfg.Panel.Password
	case "webdav.enable":
		return cfg.WebDAV.Enable
	case "webdav.username":
		return cfg.WebDAV.Username
	case "webdav.password":
		return cfg.WebDAV.Password
	case "enable.caddy":
		return cfg.Enable.Caddy
	case "enable.frps":
		return cfg.Enable.Frps
	case "enable.frpc":
		return cfg.Enable.Frpc
	case "enable.headscale":
		return cfg.Enable.Headscale
	default:
		return nil
	}
}

func set(cfg *Config, path string, value any) error {
	switch path {
	case "panel.port":
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("config %s expects string", path)
		}
		cfg.Panel.Port = v
	case "panel.path":
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("config %s expects string", path)
		}
		cfg.Panel.Path = v
	case "panel.username":
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("config %s expects string", path)
		}
		cfg.Panel.Username = v
	case "panel.password":
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("config %s expects string", path)
		}
		cfg.Panel.Password = v
	case "webdav.enable":
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("config %s expects bool", path)
		}
		cfg.WebDAV.Enable = v
	case "webdav.username":
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("config %s expects string", path)
		}
		cfg.WebDAV.Username = v
	case "webdav.password":
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("config %s expects string", path)
		}
		cfg.WebDAV.Password = v
	case "enable.caddy":
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("config %s expects bool", path)
		}
		cfg.Enable.Caddy = v
	case "enable.frps":
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("config %s expects bool", path)
		}
		cfg.Enable.Frps = v
	case "enable.frpc":
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("config %s expects bool", path)
		}
		cfg.Enable.Frpc = v
	case "enable.headscale":
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("config %s expects bool", path)
		}
		cfg.Enable.Headscale = v
	default:
		return fmt.Errorf("unknown config key: %s", path)
	}

	return nil
}

func printStartupInfo(cfg Config) {
	colorGreen := "\x1b[32m"
	colorReset := "\x1b[0m"
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			} else if ipNet.IP.To4() != nil && ipNet.IP.IsGlobalUnicast() {
				fmt.Printf("gopanel started on https://%v%v/%v\n", ipNet.IP, cfg.Panel.Port, cfg.Panel.Path)
			} else if ipNet.IP.To16() != nil && ipNet.IP.IsGlobalUnicast() {
				fmt.Printf("gopanel started on https://[%v]%v/%v\n", ipNet.IP, cfg.Panel.Port, cfg.Panel.Path)
			}
		}
	}
	fmt.Printf("Panel Port: %s%s%s\n", colorGreen, cfg.Panel.Port, colorReset)
	fmt.Printf("Panel Path: %s%s%s\n", colorGreen, cfg.Panel.Path, colorReset)
	fmt.Printf("Panel Username: %s%s%s\n", colorGreen, cfg.Panel.Username, colorReset)
	fmt.Printf("Panel Password: %s%s%s\n", colorGreen, cfg.Panel.Password, colorReset)
}
