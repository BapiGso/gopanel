package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"gopanel/core/storage"
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
	initial.SchemaVersion = SchemaVersion
	content, err := encode(initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.AtomicWrite(path, content); err != nil {
		t.Fatalf("write initial config file: %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("load temp config: %v", err)
	}
	oldGlobal := global.Swap(store)

	t.Cleanup(func() {
		global.Store(oldGlobal)
	})

	return path
}

func TestDefaultSetsExpectedStructure(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version = %d", cfg.SchemaVersion)
	}

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
	directory := t.TempDir()
	path := filepath.Join(directory, FileName)
	oldGlobal := global.Load()
	t.Cleanup(func() {
		global.Store(oldGlobal)
	})

	if err := Init(directory); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatalf("stat config file: %v", err)
	} else if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("config file mode = %o, want %o", info.Mode().Perm(), 0600)
	}

	disk := readConfigFileForTest(t, path)
	if got := Snapshot(); got != disk {
		t.Fatalf("Snapshot() = %+v, want %+v", got, disk)
	}
}

func TestUpdatePersistsEachSupportedField(t *testing.T) {
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
		name   string
		want   any
		update func(*Config)
		read   func(Config) any
	}{
		{name: "panel.port", want: ":9443", update: func(cfg *Config) { cfg.Panel.Port = ":9443" }, read: func(cfg Config) any { return cfg.Panel.Port }},
		{name: "panel.path", want: "newpath", update: func(cfg *Config) { cfg.Panel.Path = "newpath" }, read: func(cfg Config) any { return cfg.Panel.Path }},
		{name: "panel.username", want: "newuser", update: func(cfg *Config) { cfg.Panel.Username = "newuser" }, read: func(cfg Config) any { return cfg.Panel.Username }},
		{name: "panel.password", want: "newpass", update: func(cfg *Config) { cfg.Panel.Password = "newpass" }, read: func(cfg Config) any { return cfg.Panel.Password }},
		{name: "webdav.enable", want: true, update: func(cfg *Config) { cfg.WebDAV.Enable = true }, read: func(cfg Config) any { return cfg.WebDAV.Enable }},
		{name: "webdav.username", want: "davnew", update: func(cfg *Config) { cfg.WebDAV.Username = "davnew" }, read: func(cfg Config) any { return cfg.WebDAV.Username }},
		{name: "webdav.password", want: "davsecret", update: func(cfg *Config) { cfg.WebDAV.Password = "davsecret" }, read: func(cfg Config) any { return cfg.WebDAV.Password }},
		{name: "enable.caddy", want: true, update: func(cfg *Config) { cfg.Enable.Caddy = true }, read: func(cfg Config) any { return cfg.Enable.Caddy }},
		{name: "enable.frps", want: true, update: func(cfg *Config) { cfg.Enable.Frps = true }, read: func(cfg Config) any { return cfg.Enable.Frps }},
		{name: "enable.frpc", want: true, update: func(cfg *Config) { cfg.Enable.Frpc = true }, read: func(cfg Config) any { return cfg.Enable.Frpc }},
		{name: "enable.headscale", want: true, update: func(cfg *Config) { cfg.Enable.Headscale = true }, read: func(cfg Config) any { return cfg.Enable.Headscale }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := useTempGlobalConfig(t, initial)

			if _, err := Update(Current().Revision, func(cfg *Config) error {
				tt.update(cfg)
				return nil
			}); err != nil {
				t.Fatalf("Update(%q) error = %v", tt.name, err)
			}

			disk := readConfigFileForTest(t, path)
			if got := tt.read(disk); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("file field for %q = %v, want %v", tt.name, got, tt.want)
			}

			if got := tt.read(Snapshot()); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("snapshot field for %q = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestUpdateRejectsInvalidConfigWithoutChangingFile(t *testing.T) {
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

	state := Current()
	if _, err := Update(state.Revision, func(cfg *Config) error {
		cfg.Panel.Port = "invalid"
		return nil
	}); err == nil {
		t.Fatal("Update should reject invalid config")
	}

	after := readConfigFileForTest(t, path)
	if after != before {
		t.Fatalf("config file changed after rejected writes: before=%+v after=%+v", before, after)
	}
	if got := Current(); got != state {
		t.Fatal("rejected update changed in-memory config")
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

	if _, err := Update(Current().Revision, func(cfg *Config) error {
		cfg.Panel.Port = ":10443"
		cfg.WebDAV.Enable = true
		cfg.Enable.Headscale = true
		return nil
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

	if _, err := Open(path); err == nil {
		t.Fatal("load() should fail for invalid JSON")
	}
}
