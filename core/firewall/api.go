package firewall

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

type ruleWrite struct {
	Network   string `json:"network" validate:"required,oneof=ipv4 ipv6"`
	Protocol  string `json:"protocol" validate:"required,oneof=tcp udp"`
	Port      uint   `json:"port" validate:"max=65535"`
	Direction string `json:"direction" validate:"required,oneof=in out"`
	Action    string `json:"action" validate:"required,oneof=accept drop"`
}

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/firewall/rules", listRules, httpapi.Operation{ID: "listFirewallRules", Summary: "List panel-managed firewall rules; unsupported platforms return not_supported", Response: []Rule{}})
	router.Handle(http.MethodPost, "/firewall/rules", createRule, httpapi.Operation{ID: "createFirewallRule", Summary: "Add a firewall rule; port 0 means any port; may affect panel connectivity", Status: http.StatusCreated, Request: ruleWrite{}, Response: []Rule{}})
	router.Handle(http.MethodDelete, "/firewall/rules/:id", deleteRule, httpapi.Operation{ID: "deleteFirewallRule", Summary: "Delete a listed rule by its platform-specific ID", Response: map[string]string{}})
}

func available() error {
	if mgr == nil {
		return httpapi.NewError(http.StatusNotImplemented, "not_supported", "Firewall management is unavailable on this platform")
	}
	return nil
}

func listRules(ctx *echo.Context) error {
	if err := available(); err != nil {
		return err
	}
	managerMutex.Lock()
	defer managerMutex.Unlock()
	return respondRules(ctx, http.StatusOK)
}

func respondRules(ctx *echo.Context, status int) error {
	rules, err := mgr.List()
	if err != nil {
		return httpapi.NewError(http.StatusServiceUnavailable, "firewall_unavailable", err.Error())
	}
	if rules == nil {
		rules = []Rule{}
	}
	return httpapi.Respond(ctx, status, rules)
}

func createRule(ctx *echo.Context) error {
	var request ruleWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if err := available(); err != nil {
		return err
	}
	managerMutex.Lock()
	defer managerMutex.Unlock()
	if err := mgr.Add(Rule{Network: request.Network, Protocol: request.Protocol, Port: uint16(request.Port), Direction: request.Direction, Action: request.Action}); err != nil {
		return httpapi.NewError(http.StatusServiceUnavailable, "firewall_unavailable", err.Error())
	}
	return respondRules(ctx, http.StatusCreated)
}

func deleteRule(ctx *echo.Context) error {
	if err := available(); err != nil {
		return err
	}
	managerMutex.Lock()
	defer managerMutex.Unlock()
	rules, err := mgr.List()
	if err != nil {
		return httpapi.NewError(http.StatusServiceUnavailable, "firewall_unavailable", err.Error())
	}
	identifier := httpapi.Param(ctx, "id")
	found := false
	for _, rule := range rules {
		if rule.ID == identifier {
			found = true
			break
		}
	}
	if !found {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Firewall rule not found")
	}
	if err := mgr.Delete(identifier); err != nil {
		return httpapi.NewError(http.StatusServiceUnavailable, "firewall_unavailable", err.Error())
	}
	return httpapi.OK(ctx, map[string]string{"id": identifier})
}
