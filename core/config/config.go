package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sync"

	jsonparser "github.com/knadh/koanf/parsers/json"
	fileprovider "github.com/knadh/koanf/providers/file"
	structprovider "github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

const filePath = "gopanel_config.json"

type Config struct {
	Panel  PanelConfig  `json:"panel" koanf:"panel"`
	WebDAV WebDAVConfig `json:"webdav" koanf:"webdav"`
	Enable EnableConfig `json:"enable" koanf:"enable"`
}

type PanelConfig struct {
	Port     string `json:"port" koanf:"port"`
	Path     string `json:"path" koanf:"path"`
	Username string `json:"username" koanf:"username"`
	Password string `json:"password" koanf:"password"`
}

type WebDAVConfig struct {
	Enable   bool   `json:"enable" koanf:"enable"`
	Username string `json:"username" koanf:"username"`
	Password string `json:"password" koanf:"password"`
}

type EnableConfig struct {
	Caddy     bool `json:"caddy" koanf:"caddy"`
	Frps      bool `json:"frps" koanf:"frps"`
	Frpc      bool `json:"frpc" koanf:"frpc"`
	Headscale bool `json:"headscale" koanf:"headscale"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	cfg  Config
	k    *koanf.Koanf
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
	if global.k == nil {
		return nil
	}
	return global.k.Get(path)
}

func String(path string) string {
	global.mu.RLock()
	defer global.mu.RUnlock()
	if global.k == nil {
		return ""
	}
	return global.k.String(path)
}

func Bool(path string) bool {
	global.mu.RLock()
	defer global.mu.RUnlock()
	if global.k == nil {
		return false
	}
	return global.k.Bool(path)
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

	k, cfg, err := loadKoanf(s.path)
	if err != nil {
		return err
	}

	s.cfg = cfg
	s.k = k
	return nil
}

func (s *Store) write(path string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.k == nil {
		return fmt.Errorf("config store is not initialized")
	}
	if !s.k.Exists(path) {
		return fmt.Errorf("unknown config key: %s", path)
	}

	current := s.k.Get(path)
	if !sameConfigValueType(current, value) {
		return fmt.Errorf("config %s expects %T", path, current)
	}

	nextKoanf, err := koanfFromConfig(s.cfg)
	if err != nil {
		return err
	}
	if err := nextKoanf.Set(path, value); err != nil {
		return err
	}

	next, err := configFromKoanf(nextKoanf)
	if err != nil {
		return err
	}
	if err := writeConfigFile(s.path, next); err != nil {
		return err
	}

	s.cfg = next
	s.k = nextKoanf
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

	nextKoanf, err := koanfFromConfig(next)
	if err != nil {
		return err
	}

	s.cfg = next
	s.k = nextKoanf
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
	data, err := koanfFromConfig(cfg)
	if err != nil {
		return err
	}
	out, err := data.Marshal(jsonparser.Parser())
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0600)
}

func generateRandomString(n int) string {
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func loadKoanf(path string) (*koanf.Koanf, Config, error) {
	k := koanf.New(".")
	if err := k.Load(fileprovider.Provider(path), jsonparser.Parser()); err != nil {
		return nil, Config{}, err
	}
	cfg, err := configFromKoanf(k)
	if err != nil {
		return nil, Config{}, err
	}
	return k, cfg, nil
}

func koanfFromConfig(cfg Config) (*koanf.Koanf, error) {
	k := koanf.New(".")
	if err := k.Load(structprovider.Provider(cfg, "koanf"), nil); err != nil {
		return nil, err
	}
	return k, nil
}

func configFromKoanf(k *koanf.Koanf) (Config, error) {
	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func sameConfigValueType(current, next any) bool {
	if current == nil || next == nil {
		return current == next
	}
	return fmt.Sprintf("%T", current) == fmt.Sprintf("%T", next)
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
