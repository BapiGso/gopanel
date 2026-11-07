//go:build linux || darwin || freebsd

package headscale

import (
	"errors"
	"net/http"

	"github.com/juanfont/headscale/hscontrol"
	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

func currentStatus() statusView {
	state := "stopped"
	if hsRunning {
		state = "running"
	} else if hsError != "" {
		state = "failed"
	}
	return statusView{State: state, LastError: hsError, StopSupported: false}
}

func getStatus(ctx *echo.Context) error {
	hsMutex.Lock()
	defer hsMutex.Unlock()
	return httpapi.OK(ctx, currentStatus())
}

func start(ctx *echo.Context) error {
	hsMutex.Lock()
	defer hsMutex.Unlock()
	if hsRunning {
		return httpapi.Respond(ctx, http.StatusAccepted, currentStatus())
	}
	document, err := configurationFile.Read()
	if err != nil {
		return err
	}
	settings, err := parseConfig(document.Content)
	if err != nil {
		return err
	}
	request := &headscaleConfig{
		ServerURL: settings.ServerURL, ListenAddr: settings.ListenAddr, MetricsListenAddr: settings.MetricsListenAddr,
		PrivateKeyPath: settings.PrivateKeyPath, IPv4Prefix: settings.IPv4Prefix, IPv6Prefix: settings.IPv6Prefix, BaseDomain: settings.BaseDomain,
		STUNEnabled: settings.STUNEnabled, STUNListenAddr: settings.STUNListenAddr, DERPEnabled: settings.DERPEnabled,
		DERPRegionID: settings.DERPRegionID, DERPRegionCode: settings.DERPRegionCode, DERPRegionName: settings.DERPRegionName,
	}
	if settings.DERPVerifyClients {
		request.DERPVerifyClient = "on"
	}
	configuration, err := loadServerConfig(request)
	if err != nil {
		return httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", err.Error())
	}
	app, err := hscontrol.NewHeadscale(configuration)
	if err != nil {
		return httpapi.NewError(http.StatusUnprocessableEntity, "start_failed", err.Error())
	}
	var standaloneSTUN *standaloneSTUNServer
	if request.shouldStartStandaloneSTUN() {
		standaloneSTUN, err = startStandaloneSTUN(request.stunListenAddr())
		if err != nil {
			return httpapi.NewError(http.StatusUnprocessableEntity, "start_failed", err.Error())
		}
	}
	hsApp = app
	hsRunning = true
	hsError = ""
	stunSrv = standaloneSTUN
	go func() {
		err := app.Serve()
		hsMutex.Lock()
		defer hsMutex.Unlock()
		hsRunning = false
		hsApp = nil
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			hsError = err.Error()
		}
		stopStandaloneSTUNLocked()
	}()
	return httpapi.Respond(ctx, http.StatusAccepted, currentStatus())
}

func stop(ctx *echo.Context) error {
	hsMutex.Lock()
	defer hsMutex.Unlock()
	if hsRunning {
		return httpapi.NewError(http.StatusNotImplemented, "not_supported", "Upstream Headscale does not expose a safe embedded shutdown; restart the panel instead. No stop signal was sent")
	}
	return httpapi.OK(ctx, currentStatus())
}
