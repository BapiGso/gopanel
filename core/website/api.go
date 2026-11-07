package website

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/httpapi"
)

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/caddy/config", getConfig, httpapi.Operation{ID: "getCaddyConfig", Summary: "Read the native Caddyfile and its revision", Response: httpapi.ConfigDocument{}})
	router.Handle(http.MethodPut, "/caddy/config", saveConfig, httpapi.Operation{ID: "saveCaddyConfig", Summary: "Save a Caddyfile without applying it; revision is required", Request: httpapi.ConfigWrite{}, Response: httpapi.ConfigDocument{}})
	router.Handle(http.MethodGet, "/caddy/status", getStatus, httpapi.Operation{ID: "getCaddyStatus", Summary: "Read embedded Caddy state and the separate autostart preference", Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPatch, "/caddy/settings", saveSettings, httpapi.Operation{ID: "setCaddyAutostart", Summary: "Set autostart using settings_revision from status, without changing the running service", Request: httpapi.AutostartWrite{}, Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPost, "/caddy/start", start, httpapi.Operation{ID: "startCaddy", Summary: "Load the saved Caddyfile into the panel's embedded Caddy instance; admin API stays disabled", Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPost, "/caddy/reload", start, httpapi.Operation{ID: "reloadCaddy", Summary: "Apply the saved Caddyfile to the same embedded instance", Response: httpapi.ServiceStatus{}})
	router.Handle(http.MethodPost, "/caddy/stop", stop, httpapi.Operation{ID: "stopCaddy", Summary: "Stop only embedded Caddy, never the panel process", Response: httpapi.ServiceStatus{}})
}

func getConfig(ctx *echo.Context) error {
	document, err := caddyFile.Read()
	if err != nil {
		return err
	}
	return httpapi.OK(ctx, document)
}

func saveConfig(ctx *echo.Context) error {
	var request httpapi.ConfigWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if _, err := adaptConfig(*request.Content); err != nil {
		return err
	}
	document, err := caddyFile.Save(*request.Content, request.Revision)
	if err != nil {
		return err
	}
	return httpapi.OK(ctx, document)
}

func getStatus(ctx *echo.Context) error {
	current, err := status()
	if err != nil {
		return err
	}
	return httpapi.OK(ctx, current)
}

func saveSettings(ctx *echo.Context) error {
	var request httpapi.AutostartWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if _, err := config.Update(request.Revision, func(settings *config.Config) error {
		settings.Enable.Caddy = *request.Autostart
		return nil
	}); err != nil {
		return err
	}
	return getStatus(ctx)
}

func start(ctx *echo.Context) error {
	if err := caddyStart(); err != nil {
		return err
	}
	return getStatus(ctx)
}

func stop(ctx *echo.Context) error {
	if err := caddyStop(); err != nil {
		return err
	}
	return getStatus(ctx)
}
