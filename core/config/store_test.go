package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gopanel/core/storage"
)

const legacyTestConfig = `{"panel":{"port":":8443","path":"login","username":"existing-user","password":"existing-password"},"webdav":{"enable":false},"enable":{"caddy":true}}`

func openTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return store, path
}

func TestLegacyConfigMigratesOnlyOnSuccessfulSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if err := storage.AtomicWrite(path, []byte(legacyTestConfig)); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	state := store.Snapshot()
	if state.Config.SchemaVersion != SchemaVersion || !state.Config.Enable.Caddy || state.Config.Enable.Frps || state.Config.Enable.Frpc || state.Config.WebDAV.Enable {
		t.Fatal("legacy configuration was not migrated with safe defaults")
	}
	original, err := os.ReadFile(path)
	if err != nil || string(original) != legacyTestConfig {
		t.Fatal("loading legacy config rewrote the file")
	}
	updated, err := store.Update(state.Revision, func(next *Config) error {
		next.Panel.Port = ":9443"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	saved := readConfigFileForTest(t, path)
	if saved.SchemaVersion != SchemaVersion || saved.Panel.Password != "existing-password" || saved.Panel.Username != "existing-user" || saved.Panel.Port != ":9443" {
		t.Fatal("saving migrated config changed existing credentials")
	}
	reopened, err := Open(path)
	if err != nil || reopened.Snapshot() != updated {
		t.Fatalf("reopening changed the saved configuration: %v", err)
	}
}

func TestOpenRejectsInvalidDocumentsWithoutOverwritingThem(t *testing.T) {
	cases := map[string]string{
		"broken JSON":      "{",
		"null object":      "null",
		"array":            "[]",
		"multiple objects": legacyTestConfig + "{}",
		"missing account":  `{"panel":{"port":":8443","path":"login"}}`,
		"unknown field":    strings.Replace(legacyTestConfig, `"path":"login"`, `"path":"login","unknown":true`, 1),
		"invalid port":     strings.Replace(legacyTestConfig, ":8443", ":0", 1),
		"reserved path":    strings.Replace(legacyTestConfig, "login", "api", 1),
		"string boolean":   strings.Replace(legacyTestConfig, `"caddy":true`, `"caddy":"true"`, 1),
		"null boolean":     strings.Replace(legacyTestConfig, `"caddy":true`, `"caddy":null`, 1),
		"null section":     strings.Replace(legacyTestConfig, `{"enable":false}`, "null", 1),
		"missing DAV auth": strings.Replace(legacyTestConfig, `"enable":false`, `"enable":true`, 1),
		"future schema":    `{"schema_version":2,` + strings.TrimPrefix(legacyTestConfig, "{"),
		"negative schema":  `{"schema_version":-1,` + strings.TrimPrefix(legacyTestConfig, "{"),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), FileName)
			if err := storage.AtomicWrite(path, []byte(content)); err != nil {
				t.Fatal(err)
			}
			_, err := Open(path)
			var failure *ValidationError
			if !errors.As(err, &failure) {
				t.Fatalf("Open() error = %v, want ValidationError", err)
			}
			actual, err := os.ReadFile(path)
			if err != nil || string(actual) != content {
				t.Fatal("opening invalid config overwrote it")
			}
		})
	}
}

func TestUpdateErrorsPreserveSnapshotAndFile(t *testing.T) {
	store, path := openTestStore(t)
	state := store.Snapshot()
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cancelled := errors.New("cancelled update")
	cases := map[string]func(*Config) error{
		"invalid port":    func(next *Config) error { next.Panel.Port = "invalid"; return nil },
		"invalid path":    func(next *Config) error { next.Panel.Path = "admin"; return nil },
		"empty password":  func(next *Config) error { next.Panel.Password = ""; return nil },
		"invalid version": func(next *Config) error { next.SchemaVersion = 2; return nil },
		"invalid WebDAV": func(next *Config) error {
			next.WebDAV = WebDAVConfig{Enable: true}
			return nil
		},
		"callback error": func(next *Config) error {
			next.Panel.Username = "do-not-publish"
			return cancelled
		},
	}
	for name, update := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := store.Update(state.Revision, update); err == nil {
				t.Fatal("invalid update succeeded")
			}
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, original) || store.Snapshot() != state {
				t.Fatal("failed update changed the file or in-memory state")
			}
		})
	}
	copy := store.Snapshot()
	copy.Config.Panel.Password = "modified-copy"
	if store.Snapshot() != state {
		t.Fatal("mutating a returned snapshot changed the store")
	}
}

func TestUpdateIOFailureDoesNotPublishCandidate(t *testing.T) {
	store, path := openTestStore(t)
	state := store.Snapshot()
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	store.file = &storage.File{Path: t.TempDir()}
	if _, err := store.Update(state.Revision, func(next *Config) error {
		next.Panel.Username = "do-not-publish"
		return nil
	}); err == nil {
		t.Fatal("update with an inaccessible backing file succeeded")
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, original) || store.Snapshot() != state {
		t.Fatal("I/O failure changed state or the original file")
	}
}

func TestConcurrentStoresRequireRereadBeforeRetry(t *testing.T) {
	first, path := openTestStore(t)
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	initial := first.Snapshot()
	var writers sync.WaitGroup
	failures := make(chan error, 2)
	for index, store := range []*Store{first, second} {
		writers.Go(func() {
			_, err := store.Update(initial.Revision, func(next *Config) error {
				next.Panel.Port = []string{":9443", ":10443"}[index]
				return nil
			})
			failures <- err
		})
	}
	writers.Wait()
	close(failures)
	successes, conflicts := 0, 0
	for err := range failures {
		if err == nil {
			successes++
		} else if errors.Is(err, storage.ErrRevisionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes = %d, conflicts = %d", successes, conflicts)
	}
	latest, err := first.Read()
	if err != nil {
		t.Fatal(err)
	}
	other, err := second.Read()
	if err != nil || latest != other || latest.Revision == initial.Revision {
		t.Fatal("reread did not synchronize both stores")
	}
	if _, err := second.Update(initial.Revision, func(next *Config) error { return nil }); !errors.Is(err, storage.ErrRevisionConflict) {
		t.Fatalf("stale retry error = %v", err)
	}
}

func TestExternalEditsRequireRereadAndInvalidEditsKeepLastValidSnapshot(t *testing.T) {
	store, path := openTestStore(t)
	initial := store.Snapshot()
	external := initial.Config
	external.Panel.Username = "external-user"
	content, err := encode(external)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.AtomicWrite(path, content); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(initial.Revision, func(next *Config) error {
		next.Enable.Caddy = true
		return nil
	}); !errors.Is(err, storage.ErrRevisionConflict) {
		t.Fatalf("unobserved external edit was overwritten: %v", err)
	}
	if store.Snapshot() != initial {
		t.Fatal("failed update published a new snapshot")
	}
	latest, err := store.Read()
	if err != nil || latest.Config.Panel.Username != "external-user" || latest.Revision == initial.Revision {
		t.Fatal("reread did not load the external edit")
	}
	if err := storage.AtomicWrite(path, []byte("{broken")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(); err == nil || store.Snapshot() != latest {
		t.Fatal("invalid external config replaced the last valid snapshot")
	}
}

func TestConfigPathIsResolvedOnce(t *testing.T) {
	firstDirectory := t.TempDir()
	secondDirectory := t.TempDir()
	t.Chdir(firstDirectory)
	store, err := Open(FileName)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(secondDirectory)
	if _, err := store.Update(store.Snapshot().Revision, func(next *Config) error {
		next.Panel.Port = ":9443"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(secondDirectory, FileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("changing working directory redirected a config write")
	}
	if saved := readConfigFileForTest(t, filepath.Join(firstDirectory, FileName)); saved.Panel.Port != ":9443" {
		t.Fatal("the originally selected configuration was not updated")
	}
}

func TestInitializationIsExplicitAndFailureKeepsPreviousStore(t *testing.T) {
	previous := global.Swap(nil)
	t.Cleanup(func() { global.Store(previous) })
	if _, err := Read(); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("uninitialized Read = %v", err)
	}
	if _, err := Update("", func(next *Config) error { return nil }); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("uninitialized Update = %v", err)
	}
	if err := Init(""); err == nil {
		t.Fatal("initialization without a directory succeeded")
	}
	if err := Init(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	valid := Current()
	directory := t.TempDir()
	if err := storage.AtomicWrite(filepath.Join(directory, FileName), []byte("{broken")); err != nil {
		t.Fatal(err)
	}
	if err := Init(directory); err == nil || Current() != valid {
		t.Fatal("failed initialization replaced the previous store")
	}
}
