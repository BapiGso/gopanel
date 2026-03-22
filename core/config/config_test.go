package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readConfigFileForTest(t *testing.T, path string) Config {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config file: %v", err)
	}

	return cfg
}

func useTempGlobalConfig(t *testing.T, initial Config) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "gopanel_config.json")
	if err := writeConfigFile(path, initial); err != nil {
		t.Fatalf("write initial config file: %v", err)
	}

	oldGlobal := global
	global = &Store{path: path}
	if err := global.load(); err != nil {
		t.Fatalf("load temp config: %v", err)
	}

	t.Cleanup(func() {
		global = oldGlobal
	})

	return path
}

func TestDefaultSetsExpectedStructure(t *testing.T) {
	cfg := Default()

	if cfg.Panel.Port != ":8443" {
		t.Fatalf("Panel.Port = %q, want %q", cfg.Panel.Port, ":8443")
	}
	if len(cfg.Panel.Path) != 8 {
		t.Fatalf("len(Panel.Path) = %d, want 8", len(cfg.Panel.Path))
	}
	if len(cfg.Panel.Username) != 12 {
		t.Fatalf("len(Panel.Username) = %d, want 12", len(cfg.Panel.Username))
	}
	if len(cfg.Panel.Password) != 12 {
		t.Fatalf("len(Panel.Password) = %d, want 12", len(cfg.Panel.Password))
	}
	if cfg.WebDAV.Enable {
		t.Fatal("WebDAV.Enable = true, want false")
	}
	if len(cfg.WebDAV.Username) != 6 {
		t.Fatalf("len(WebDAV.Username) = %d, want 6", len(cfg.WebDAV.Username))
	}
	if len(cfg.WebDAV.Password) != 6 {
		t.Fatalf("len(WebDAV.Password) = %d, want 6", len(cfg.WebDAV.Password))
	}
	if cfg.Enable.Caddy || cfg.Enable.Frps || cfg.Enable.Frpc || cfg.Enable.Headscale {
		t.Fatal("default service flags should all be false")
	}
}

func TestInitCreatesDefaultConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gopanel_config.json")
	oldGlobal := global
	global = &Store{path: path}
	t.Cleanup(func() {
		global = oldGlobal
	})

	if err := Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}

	disk := readConfigFileForTest(t, path)
	if got := Snapshot(); got != disk {
		t.Fatalf("Snapshot() = %+v, want %+v", got, disk)
	}
}

func TestWritePersistsEachSupportedKey(t *testing.T) {
	initial := Config{
		Panel: PanelConfig{
			Port:     ":8443",
			Path:     "oldpath",
			Username: "olduser",
			Password: "oldpass",
		},
		WebDAV: WebDAVConfig{
			Enable:   false,
			Username: "davold",
			Password: "davpwd",
		},
		Enable: EnableConfig{},
	}

	tests := []struct {
		name string
		path string
		want any
		read func(Config) any
	}{
		{name: "panel.port", path: "panel.port", want: ":9443", read: func(cfg Config) any { return cfg.Panel.Port }},
		{name: "panel.path", path: "panel.path", want: "newpath", read: func(cfg Config) any { return cfg.Panel.Path }},
		{name: "panel.username", path: "panel.username", want: "newuser", read: func(cfg Config) any { return cfg.Panel.Username }},
		{name: "panel.password", path: "panel.password", want: "newpass", read: func(cfg Config) any { return cfg.Panel.Password }},
		{name: "webdav.enable", path: "webdav.enable", want: true, read: func(cfg Config) any { return cfg.WebDAV.Enable }},
		{name: "webdav.username", path: "webdav.username", want: "davnew", read: func(cfg Config) any { return cfg.WebDAV.Username }},
		{name: "webdav.password", path: "webdav.password", want: "davsecret", read: func(cfg Config) any { return cfg.WebDAV.Password }},
		{name: "enable.caddy", path: "enable.caddy", want: true, read: func(cfg Config) any { return cfg.Enable.Caddy }},
		{name: "enable.frps", path: "enable.frps", want: true, read: func(cfg Config) any { return cfg.Enable.Frps }},
		{name: "enable.frpc", path: "enable.frpc", want: true, read: func(cfg Config) any { return cfg.Enable.Frpc }},
		{name: "enable.headscale", path: "enable.headscale", want: true, read: func(cfg Config) any { return cfg.Enable.Headscale }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := useTempGlobalConfig(t, initial)

			if err := Write(tt.path, tt.want); err != nil {
				t.Fatalf("Write(%q, %v) error = %v", tt.path, tt.want, err)
			}

			disk := readConfigFileForTest(t, path)
			if got := tt.read(disk); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("file field for %q = %v, want %v", tt.path, got, tt.want)
			}

			if got := Get(tt.path); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Get(%q) = %v, want %v", tt.path, got, tt.want)
			}

			switch want := tt.want.(type) {
			case string:
				if got := String(tt.path); got != want {
					t.Fatalf("String(%q) = %q, want %q", tt.path, got, want)
				}
			case bool:
				if got := Bool(tt.path); got != want {
					t.Fatalf("Bool(%q) = %v, want %v", tt.path, got, want)
				}
			}
		})
	}
}

func TestWriteRejectsInvalidTypeAndUnknownKeyWithoutChangingFile(t *testing.T) {
	initial := Config{
		Panel: PanelConfig{
			Port:     ":8443",
			Path:     "panelpath",
			Username: "paneluser",
			Password: "panelpass",
		},
		WebDAV: WebDAVConfig{
			Enable:   false,
			Username: "davuser",
			Password: "davpass",
		},
		Enable: EnableConfig{
			Caddy: true,
		},
	}

	path := useTempGlobalConfig(t, initial)
	before := readConfigFileForTest(t, path)

	if err := Write("panel.port", true); err == nil {
		t.Fatal("Write should reject invalid value type")
	}
	if err := Write("unknown.key", "value"); err == nil {
		t.Fatal("Write should reject unknown key")
	}

	after := readConfigFileForTest(t, path)
	if after != before {
		t.Fatalf("config file changed after rejected writes: before=%+v after=%+v", before, after)
	}
	if got := Get("unknown.key"); got != nil {
		t.Fatalf("Get(unknown.key) = %v, want nil", got)
	}
}

func TestUpdatePersistsMultipleFields(t *testing.T) {
	initial := Config{
		Panel: PanelConfig{
			Port:     ":8443",
			Path:     "panelpath",
			Username: "paneluser",
			Password: "panelpass",
		},
		WebDAV: WebDAVConfig{
			Enable:   false,
			Username: "davuser",
			Password: "davpass",
		},
		Enable: EnableConfig{},
	}

	path := useTempGlobalConfig(t, initial)

	if err := Update(func(cfg *Config) {
		cfg.Panel.Port = ":10443"
		cfg.WebDAV.Enable = true
		cfg.Enable.Headscale = true
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	disk := readConfigFileForTest(t, path)
	if disk.Panel.Port != ":10443" {
		t.Fatalf("Panel.Port = %q, want %q", disk.Panel.Port, ":10443")
	}
	if !disk.WebDAV.Enable {
		t.Fatal("WebDAV.Enable = false, want true")
	}
	if !disk.Enable.Headscale {
		t.Fatal("Enable.Headscale = false, want true")
	}
	if got := Snapshot(); got != disk {
		t.Fatalf("Snapshot() = %+v, want %+v", got, disk)
	}
}

func TestStoreLoadRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken_config.json")
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatalf("write broken config file: %v", err)
	}

	store := &Store{path: path}
	if err := store.load(); err == nil {
		t.Fatal("load() should fail for invalid JSON")
	}
}
