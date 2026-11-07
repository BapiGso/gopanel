package frp

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	frpconfig "github.com/fatedier/frp/pkg/config"
	configsource "github.com/fatedier/frp/pkg/config/source"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/server"

	"gopanel/core/config"
	"gopanel/core/httpapi"
)

type serviceRuntime struct {
	server    *server.Service
	client    *client.Service
	done      chan struct{}
	state     string
	revision  string
	lastError string
}

var (
	serviceMutex sync.Mutex
	services     = map[string]*serviceRuntime{
		"frps": {state: "stopped"},
		"frpc": {state: "stopped"},
	}
)

func validateServer(path string) error {
	configuration, _, err := frpconfig.LoadServerConfig(path, true)
	if err != nil {
		return err
	}
	checker := validation.NewConfigValidator(security.NewUnsafeFeatures([]string{}))
	_, err = checker.ValidateServerConfig(configuration)
	return err
}

func validateClient(path string) error {
	configuration, proxies, visitors, _, err := frpconfig.LoadClientConfig(path, true)
	if err != nil {
		return err
	}
	_, err = validation.ValidateAllClientConfig(configuration, proxies, visitors, nil)
	return err
}

func temporaryConfiguration(content string) (string, func(), error) {
	temporary, err := os.CreateTemp(".", ".gopanel-frp-*.conf")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.Remove(temporary.Name()) }
	if _, err := temporary.WriteString(content); err != nil {
		temporary.Close()
		cleanup()
		return "", nil, err
	}
	if err := temporary.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return temporary.Name(), cleanup, nil
}

func validatePath(name, path string) error {
	var err error
	if name == "frps" {
		err = validateServer(path)
	} else {
		err = validateClient(path)
	}
	if err != nil {
		return httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", err.Error())
	}
	return nil
}

func validateContent(name, content string) error {
	path, cleanup, err := temporaryConfiguration(content)
	if err != nil {
		return err
	}
	defer cleanup()
	return validatePath(name, path)
}

func startService(name string, reload bool) error {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()
	current := services[name]
	if current.state == "running" && !reload {
		return nil
	}
	document, err := configFiles[name].Read()
	if err != nil {
		return err
	}
	configurationPath, cleanup, err := temporaryConfiguration(document.Content)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := validatePath(name, configurationPath); err != nil {
		return err
	}
	if err := stopLocked(current); err != nil {
		return err
	}
	next := &serviceRuntime{state: "running", revision: document.Revision, done: make(chan struct{})}
	if name == "frps" {
		configuration, _, err := frpconfig.LoadServerConfig(configurationPath, true)
		if err != nil {
			return err
		}
		log.InitLogger(configuration.Log.To, configuration.Log.Level, int(configuration.Log.MaxDays), configuration.Log.DisablePrintColor)
		next.server, err = server.NewService(configuration)
		if err != nil {
			return httpapi.NewError(http.StatusUnprocessableEntity, "start_failed", err.Error())
		}
	} else {
		configuration, proxies, visitors, _, err := frpconfig.LoadClientConfig(configurationPath, true)
		if err != nil {
			return err
		}
		log.InitLogger(configuration.Log.To, configuration.Log.Level, int(configuration.Log.MaxDays), configuration.Log.DisablePrintColor)
		source := configsource.NewConfigSource()
		if err := source.ReplaceAll(proxies, visitors); err != nil {
			return err
		}
		next.client, err = client.NewService(client.ServiceOptions{
			Common: configuration, ConfigSourceAggregator: configsource.NewAggregator(source),
		})
		if err != nil {
			return httpapi.NewError(http.StatusUnprocessableEntity, "start_failed", err.Error())
		}
	}
	services[name] = next
	go func() {
		var runError error
		if next.server != nil {
			next.server.Run(context.Background())
		} else {
			runError = next.client.Run(context.Background())
		}
		close(next.done)
		serviceMutex.Lock()
		defer serviceMutex.Unlock()
		if services[name] != next {
			return
		}
		if runError != nil && next.state != "stopping" && next.state != "stopped" {
			next.state = "failed"
			next.lastError = runError.Error()
		} else {
			next.state = "stopped"
		}
		next.server = nil
		next.client = nil
	}()
	return nil
}

func stopLocked(current *serviceRuntime) error {
	if current.done == nil {
		current.state = "stopped"
		return nil
	}
	current.state = "stopping"
	if current.server != nil {
		if err := current.server.Close(); err != nil {
			return err
		}
	}
	if current.client != nil {
		current.client.GracefulClose(500 * time.Millisecond)
	}
	select {
	case <-current.done:
		current.state = "stopped"
		current.server = nil
		current.client = nil
		current.done = nil
		current.revision = ""
		return nil
	case <-time.After(5 * time.Second):
		return httpapi.NewError(http.StatusServiceUnavailable, "service_stopping", "Service is still stopping; poll its status before retrying")
	}
}

func serviceStatus(name string) (httpapi.ServiceStatus, error) {
	settings, err := config.Read()
	if err != nil {
		return httpapi.ServiceStatus{}, err
	}
	serviceMutex.Lock()
	defer serviceMutex.Unlock()
	document, err := configFiles[name].Read()
	if err != nil {
		return httpapi.ServiceStatus{}, err
	}
	current := services[name]
	autostart := settings.Config.Enable.Frps
	if name == "frpc" {
		autostart = settings.Config.Enable.Frpc
	}
	return httpapi.ServiceStatus{
		State: current.state, Autostart: autostart, AppliedRevision: current.revision,
		SettingsRevision: settings.Revision,
		PendingChanges:   current.revision != document.Revision, LastError: current.lastError,
	}, nil
}
