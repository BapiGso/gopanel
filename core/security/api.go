package security

import (
	"net/http"
	"runtime"
	"time"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/httpapi"
)

type settingsWrite struct {
	Revision string  `json:"revision" validate:"required,len=64,hexadecimal"`
	Port     *string `json:"port" validate:"omitempty,min=1,max=200"`
	Path     *string `json:"path" validate:"omitempty,min=1,max=200"`
	Username *string `json:"username" validate:"omitempty,min=1,max=200"`
	Password *string `json:"password" validate:"omitempty,min=8,max=200"`
}

type settingsView struct {
	Revision           string `json:"revision"`
	Port               string `json:"port"`
	Path               string `json:"path"`
	Username           string `json:"username"`
	PasswordConfigured bool   `json:"password_configured"`
	RestartRequired    bool   `json:"restart_required"`
}

type maintenanceResult struct {
	State           string `json:"state"`
	RestartRequired bool   `json:"restart_required"`
}

func RegisterAPI(router *httpapi.Router) {
	running := config.Snapshot().Panel
	router.Handle(http.MethodGet, "/panel/settings", func(ctx *echo.Context) error {
		return respondSettings(ctx, running)
	}, httpapi.Operation{ID: "getPanelSettings", Summary: "Read stored panel settings and whether a restart is pending, without exposing its password", Response: settingsView{}})
	router.Handle(http.MethodPatch, "/panel/settings", func(ctx *echo.Context) error {
		return patchSettings(ctx, running)
	}, httpapi.Operation{ID: "setPanelSettings", Summary: "Patch stored settings using the last read revision; port/path changes require a separate restart", Request: settingsWrite{}, Response: settingsView{}})
	router.Handle(http.MethodPost, "/panel/update", updatePanel, httpapi.Operation{ID: "updatePanel", Summary: "Replace the panel binary with the latest release; does not restart it", Response: maintenanceResult{}})
	router.Handle(http.MethodPost, "/panel/restart", restartPanel, httpapi.Operation{ID: "restartPanel", Summary: "Restart the entire panel on Linux; other platforms return not_supported", Status: http.StatusAccepted, Response: maintenanceResult{}})
}

func respondSettings(ctx *echo.Context, running config.PanelConfig) error {
	state, err := config.Read()
	if err != nil {
		return err
	}
	return settingsResponse(ctx, state, running)
}

func settingsResponse(ctx *echo.Context, state config.State, running config.PanelConfig) error {
	settings := state.Config.Panel
	restartRequired := settings.Port != running.Port || settings.Path != running.Path
	return httpapi.OK(ctx, settingsView{
		Revision: state.Revision,
		Port:     settings.Port, Path: settings.Path, Username: settings.Username,
		PasswordConfigured: settings.Password != "", RestartRequired: restartRequired,
	})
}

func patchSettings(ctx *echo.Context, running config.PanelConfig) error {
	var request settingsWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if request.Port == nil && request.Path == nil && request.Username == nil && request.Password == nil {
		return httpapi.NewError(http.StatusUnprocessableEntity, "validation_failed", "Provide at least one setting")
	}
	state, err := config.Update(request.Revision, func(settings *config.Config) error {
		if request.Port != nil {
			settings.Panel.Port = *request.Port
		}
		if request.Path != nil {
			settings.Panel.Path = *request.Path
		}
		if request.Username != nil {
			settings.Panel.Username = *request.Username
		}
		if request.Password != nil {
			settings.Panel.Password = *request.Password
		}
		return nil
	})
	if err != nil {
		return err
	}
	return settingsResponse(ctx, state, running)
}

func updatePanel(ctx *echo.Context) error {
	if _, known := currentVersion(); !known {
		return httpapi.NewError(http.StatusConflict, "version_unavailable", "Updates require a release build with a known version")
	}
	if err := updateBinaryIfNeeded(); err != nil {
		return httpapi.NewError(http.StatusBadGateway, "update_failed", err.Error())
	}
	return httpapi.OK(ctx, maintenanceResult{State: "updated", RestartRequired: true})
}

func restartPanel(ctx *echo.Context) error {
	if runtime.GOOS != "linux" {
		return httpapi.NewError(http.StatusNotImplemented, "not_supported", "Restart the panel manually on this operating system")
	}
	if err := httpapi.Respond(ctx, http.StatusAccepted, maintenanceResult{State: "restarting"}); err != nil {
		return err
	}
	_ = http.NewResponseController(ctx.Response()).Flush()
	logger := ctx.Logger()
	go func() {
		time.Sleep(300 * time.Millisecond)
		if err := restart(); err != nil {
			logger.Error("panel restart failed", "error", err)
		}
	}()
	return nil
}
