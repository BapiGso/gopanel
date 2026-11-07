package frp

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/httpapi"
)

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/frp/:service/config", getConfig, httpapi.Operation{ID: "getFRPConfig", Summary: "Read native FRP config; service must be frps or frpc", Response: httpapi.ConfigDocument{}})
	router.Handle(http.MethodPut, "/frp/:service/config", saveConfig, httpapi.Operation{ID: "saveFRPConfig", Summary: "Validate and save native FRP config without applying it", Request: httpapi.ConfigWrite{}, Response: httpapi.ConfigDocument{}})
	router.Handle(http.MethodGet, "/frp/:service/status", getStatus, httpapi.Operation{ID: "getFRPStatus", Summary: "Read service-loop state, not remote connectivity readiness; service is frps or frpc", Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPatch, "/frp/:service/settings", saveSettings, httpapi.Operation{ID: "setFRPAutostart", Summary: "Set autostart for frps or frpc using settings_revision from status", Request: httpapi.AutostartWrite{}, Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPost, "/frp/:service/start", start, httpapi.Operation{ID: "startFRP", Summary: "Start the saved frps or frpc config asynchronously; already running is a no-op", Status: http.StatusAccepted, Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPost, "/frp/:service/reload", reload, httpapi.Operation{ID: "reloadFRP", Summary: "Validate saved config, then stop and replace the embedded FRP instance", Status: http.StatusAccepted, Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPost, "/frp/:service/stop", stop, httpapi.Operation{ID: "stopFRP", Summary: "Stop only the embedded frps or frpc instance", Response: httpapi.ServiceStatus{}})
}

func serviceName(ctx *echo.Context) (string, error) {
	name := ctx.Param("service")
	if _, exists := configFiles[name]; !exists {
		return "", httpapi.NewError(http.StatusNotFound, "not_found", "FRP service must be frps or frpc")
	}
	return name, nil
}

func getConfig(ctx *echo.Context) error {
	name, err := serviceName(ctx)
	if err != nil {
		return err
	}
	document, err := configFiles[name].Read()
	if err != nil {
		return err
	}
	return httpapi.OK(ctx, document)
}

func saveConfig(ctx *echo.Context) error {
	name, err := serviceName(ctx)
	if err != nil {
		return err
	}
	var request httpapi.ConfigWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if err := validateContent(name, *request.Content); err != nil {
		return err
	}
	document, err := configFiles[name].Save(*request.Content, request.Revision)
	if err != nil {
		return err
	}
	return httpapi.OK(ctx, document)
}

func getStatus(ctx *echo.Context) error {
	return respondStatus(ctx, http.StatusOK)
}

func respondStatus(ctx *echo.Context, statusCode int) error {
	name, err := serviceName(ctx)
	if err != nil {
		return err
	}
	current, err := serviceStatus(name)
	if err != nil {
		return err
	}
	return httpapi.Respond(ctx, statusCode, current)
}

func saveSettings(ctx *echo.Context) error {
	name, err := serviceName(ctx)
	if err != nil {
		return err
	}
	var request httpapi.AutostartWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if _, err := config.Update(request.Revision, func(settings *config.Config) error {
		if name == "frps" {
			settings.Enable.Frps = *request.Autostart
		} else {
			settings.Enable.Frpc = *request.Autostart
		}
		return nil
	}); err != nil {
		return err
	}
	return getStatus(ctx)
}

func start(ctx *echo.Context) error {
	return run(ctx, false)
}

func reload(ctx *echo.Context) error {
	return run(ctx, true)
}

func run(ctx *echo.Context, reload bool) error {
	name, err := serviceName(ctx)
	if err != nil {
		return err
	}
	if err := startService(name, reload); err != nil {
		return err
	}
	return respondStatus(ctx, http.StatusAccepted)
}

func stop(ctx *echo.Context) error {
	name, err := serviceName(ctx)
	if err != nil {
		return err
	}
	serviceMutex.Lock()
	err = stopLocked(services[name])
	serviceMutex.Unlock()
	if err != nil {
		return err
	}
	return getStatus(ctx)
}
