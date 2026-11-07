package headscale

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
	"gopanel/core/storage"
)

type serverConfig struct {
	ServerURL         string `json:"server_url"`
	ListenAddr        string `json:"listen_addr"`
	MetricsListenAddr string `json:"metrics_listen_addr"`
	PrivateKeyPath    string `json:"private_key_path"`
	IPv4Prefix        string `json:"ipv4_prefix"`
	IPv6Prefix        string `json:"ipv6_prefix"`
	BaseDomain        string `json:"base_domain"`
	STUNEnabled       bool   `json:"stun_enabled"`
	STUNListenAddr    string `json:"stun_listen_addr"`
	DERPEnabled       bool   `json:"derp_enabled"`
	DERPRegionID      int    `json:"derp_region_id"`
	DERPRegionCode    string `json:"derp_region_code"`
	DERPRegionName    string `json:"derp_region_name"`
	DERPVerifyClients bool   `json:"derp_verify_clients"`
}

type statusView struct {
	State         string `json:"state"`
	LastError     string `json:"last_error,omitempty"`
	StopSupported bool   `json:"stop_supported"`
}

var configurationFile = &storage.File{Path: "gopanel_headscale.json"}

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/headscale/config", getConfig, httpapi.Operation{ID: "getHeadscaleConfig", Summary: "Read the existing experimental embedded Headscale configuration as a JSON content string", Response: httpapi.ConfigDocument{}})
	router.Handle(http.MethodPut, "/headscale/config", saveConfig, httpapi.Operation{ID: "saveHeadscaleConfig", Summary: "Save experimental Headscale configuration without starting it; not the full upstream YAML schema", Request: httpapi.ConfigWrite{}, Response: httpapi.ConfigDocument{}})
	router.Handle(http.MethodGet, "/headscale/status", getStatus, httpapi.Operation{ID: "getHeadscaleStatus", Summary: "Read experimental service state; running does not guarantee startup readiness", Response: statusView{}})
	router.Handle(http.MethodPost, "/headscale/start", start, httpapi.Operation{ID: "startHeadscale", Summary: "Start saved configuration asynchronously on Linux/macOS/FreeBSD; other platforms return not_supported", Status: http.StatusAccepted, Response: statusView{}})
	router.Handle(http.MethodPost, "/headscale/stop", stop, httpapi.Operation{ID: "stopHeadscale", Summary: "Already stopped is a no-op; a running embedded Headscale cannot be stopped safely and returns not_supported", Response: statusView{}})
}

func parseConfig(content string) (*serverConfig, error) {
	var configuration serverConfig
	decoder := json.NewDecoder(bytes.NewBufferString(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", err.Error())
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", "Send exactly one configuration object")
	}
	serverURL, err := url.Parse(configuration.ServerURL)
	if err != nil || serverURL.Hostname() == "" || (serverURL.Scheme != "http" && serverURL.Scheme != "https") {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", "server_url must be an HTTP or HTTPS URL")
	}
	prefix4, err := netip.ParsePrefix(configuration.IPv4Prefix)
	if err != nil || !prefix4.Addr().Is4() {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", "ipv4_prefix must be an IPv4 CIDR")
	}
	prefix6, err := netip.ParsePrefix(configuration.IPv6Prefix)
	if err != nil || !prefix6.Addr().Is6() {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", "ipv6_prefix must be an IPv6 CIDR")
	}
	if configuration.ListenAddr == "" || configuration.PrivateKeyPath == "" || configuration.BaseDomain == "" {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_config", "listen_addr, private_key_path and base_domain are required")
	}
	return &configuration, nil
}

func getConfig(ctx *echo.Context) error {
	document, err := configurationFile.Read()
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
	if _, err := parseConfig(*request.Content); err != nil {
		return err
	}
	document, err := configurationFile.Save(*request.Content, request.Revision)
	if err != nil {
		return err
	}
	return httpapi.OK(ctx, document)
}

func Initialize(directory string) error {
	configurationFile = &storage.File{Path: filepath.Join(directory, "gopanel_headscale.json")}
	configuration := serverConfig{
		ServerURL: "http://127.0.0.1:8080", ListenAddr: "0.0.0.0:8080", MetricsListenAddr: "127.0.0.1:9090",
		PrivateKeyPath: "data/headscale/noise_private.key", IPv4Prefix: "100.64.0.0/10", IPv6Prefix: "fd7a:115c:a1e0::/48",
		BaseDomain: "example.com", STUNListenAddr: "0.0.0.0:3478", DERPRegionID: 999, DERPRegionCode: "gopanel", DERPRegionName: "GoPanel Embedded DERP",
	}
	content, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return err
	}
	return configurationFile.Ensure(content)
}
