package firewall

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v5"
)

var (
	mgr          Manager
	managerMutex sync.Mutex
)

func init() {
	manager, err := NewManager()
	if err == nil {
		mgr = manager
	}
}

func Index(ctx *echo.Context) error {
	if mgr == nil {
		return ctx.Render(http.StatusOK, "unavailable.template", nil)
	}
	managerMutex.Lock()
	defer managerMutex.Unlock()
	rules, err := mgr.List()
	if err != nil {
		return err
	}
	return ctx.Render(http.StatusOK, "firewall.template", rules)
}
