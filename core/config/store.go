package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-playground/validator/v10"

	"gopanel/core/storage"
)

var ErrNotInitialized = errors.New("configuration is not initialized; call config.Init with an explicit directory")

type State struct {
	Config   Config
	Revision string
}

type ValidationError struct {
	Cause error
}

func (failure *ValidationError) Error() string {
	return "Invalid panel configuration: " + failure.Cause.Error()
}

func (failure *ValidationError) Unwrap() error {
	return failure.Cause
}

type Store struct {
	file  *storage.File
	mu    sync.RWMutex
	state State
}

var global atomic.Pointer[Store]

var validate = func() *validator.Validate {
	checker := validator.New()
	checker.RegisterTagNameFunc(func(field reflect.StructField) string {
		return strings.Split(field.Tag.Get("json"), ",")[0]
	})
	_ = checker.RegisterValidation("listen_address", func(field validator.FieldLevel) bool {
		_, portText, err := net.SplitHostPort(field.Field().String())
		port, parseError := strconv.Atoi(portText)
		return err == nil && parseError == nil && port >= 1 && port <= 65535
	})
	_ = checker.RegisterValidation("panel_path", func(field validator.FieldLevel) bool {
		path := field.Field().String()
		return !strings.ContainsAny(path, "/?*:#%\\") && path != "api" && path != "admin" && path != "assets" && path != "webdav"
	})
	return checker
}()

func Init(directory string) error {
	if directory == "" {
		return fmt.Errorf("configuration directory is required")
	}
	store, err := Open(filepath.Join(directory, FileName))
	if err != nil {
		return err
	}
	global.Store(store)
	return nil
}

func Directory() string {
	store := global.Load()
	if store == nil {
		return ""
	}
	return filepath.Dir(store.file.Path)
}

func Snapshot() Config {
	return Current().Config
}

func Current() State {
	store := global.Load()
	if store == nil {
		panic(ErrNotInitialized)
	}
	return store.Snapshot()
}

func Read() (State, error) {
	store := global.Load()
	if store == nil {
		return State{}, ErrNotInitialized
	}
	return store.Read()
}

func Update(revision string, update func(*Config) error) (State, error) {
	store := global.Load()
	if store == nil {
		return State{}, ErrNotInitialized
	}
	return store.Update(revision, update)
}

func Open(path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
		return nil, err
	}
	file := &storage.File{Path: absolute}
	if _, err := file.Read(); errors.Is(err, os.ErrNotExist) {
		initial, err := Default()
		if err != nil {
			return nil, err
		}
		content, err := encode(initial)
		if err != nil {
			return nil, err
		}
		if err := file.Ensure(content); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	store := &Store{file: file}
	if _, err := store.Read(); err != nil {
		return nil, fmt.Errorf("read %s: %w", absolute, err)
	}
	return store, nil
}

func (store *Store) Snapshot() State {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.state
}

func (store *Store) Read() (State, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	document, err := store.file.Read()
	if err != nil {
		return State{}, err
	}
	if document.Revision == store.state.Revision {
		return store.state, nil
	}
	configuration, err := decode([]byte(document.Content))
	if err != nil {
		return State{}, err
	}
	store.state = State{Config: configuration, Revision: document.Revision}
	return store.state, nil
}

func (store *Store) Update(revision string, update func(*Config) error) (State, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if revision != store.state.Revision {
		return State{}, storage.ErrRevisionConflict
	}
	next := store.state.Config
	if err := update(&next); err != nil {
		return State{}, err
	}
	if err := next.Validate(); err != nil {
		return State{}, err
	}
	content, err := encode(next)
	if err != nil {
		return State{}, err
	}
	document, err := store.file.Save(string(content), revision)
	if err != nil {
		return State{}, err
	}
	store.state = State{Config: next, Revision: document.Revision}
	return store.state, nil
}

func (configuration Config) Validate() error {
	if err := validate.Struct(configuration); err != nil {
		return &ValidationError{Cause: err}
	}
	return nil
}

func encode(configuration Config) ([]byte, error) {
	content, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func decode(content []byte) (Config, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(content), []byte("{")) {
		return Config{}, &ValidationError{Cause: fmt.Errorf("configuration must be a JSON object")}
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var configuration Config
	if err := decoder.Decode(&configuration); err != nil {
		return Config{}, &ValidationError{Cause: err}
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Config{}, &ValidationError{Cause: fmt.Errorf("configuration must contain exactly one JSON object")}
	}
	if err := rejectNull(content, ""); err != nil {
		return Config{}, &ValidationError{Cause: err}
	}
	if configuration.SchemaVersion == 0 {
		configuration.SchemaVersion = SchemaVersion
	}
	if configuration.SchemaVersion != SchemaVersion {
		return Config{}, &ValidationError{Cause: fmt.Errorf("unsupported schema_version %d; supported version is %d", configuration.SchemaVersion, SchemaVersion)}
	}
	if err := configuration.Validate(); err != nil {
		return Config{}, err
	}
	return configuration, nil
}

func rejectNull(content []byte, prefix string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(content, &fields); err != nil {
		return err
	}
	for name, value := range fields {
		field := prefix + name
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s must not be null", field)
		}
		if bytes.HasPrefix(bytes.TrimSpace(value), []byte("{")) {
			if err := rejectNull(value, field+"."); err != nil {
				return err
			}
		}
	}
	return nil
}
