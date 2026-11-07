package webdav

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/httpapi"
)

type settingsWrite struct {
	Revision string  `json:"revision" validate:"required,len=64,hexadecimal"`
	Enable   *bool   `json:"enable"`
	Username *string `json:"username" validate:"omitempty,min=1,max=200"`
	Password *string `json:"password" validate:"omitempty,min=1,max=200"`
}

type settingsView struct {
	Revision           string `json:"revision"`
	Enable             bool   `json:"enable"`
	Username           string `json:"username"`
	PasswordConfigured bool   `json:"password_configured"`
}

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/webdav/settings", getSettings, httpapi.Operation{ID: "getWebDAVSettings", Summary: "Read WebDAV settings without exposing its password", Response: settingsView{}})
	router.Handle(http.MethodPatch, "/webdav/settings", patchSettings, httpapi.Operation{ID: "setWebDAVSettings", Summary: "Patch WebDAV settings using the last read revision; omitted fields are preserved, enable is a boolean", Request: settingsWrite{}, Response: settingsView{}})
}

func getSettings(ctx *echo.Context) error {
	state, err := config.Read()
	if err != nil {
		return err
	}
	return settingsResponse(ctx, state)
}

func settingsResponse(ctx *echo.Context, state config.State) error {
	settings := state.Config.WebDAV
	return httpapi.OK(ctx, settingsView{Revision: state.Revision, Enable: settings.Enable, Username: settings.Username, PasswordConfigured: settings.Password != ""})
}

func patchSettings(ctx *echo.Context) error {
	var request settingsWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if request.Enable == nil && request.Username == nil && request.Password == nil {
		return httpapi.NewError(http.StatusUnprocessableEntity, "validation_failed", "Provide at least one setting")
	}
	state, err := config.Update(request.Revision, func(settings *config.Config) error {
		if request.Enable != nil {
			settings.WebDAV.Enable = *request.Enable
		}
		if request.Username != nil {
			settings.WebDAV.Username = *request.Username
		}
		if request.Password != nil {
			settings.WebDAV.Password = *request.Password
		}
		return nil
	})
	if err != nil {
		return err
	}
	return settingsResponse(ctx, state)
}
