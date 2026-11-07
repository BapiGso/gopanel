package term

import (
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

type terminalCreate struct {
	Host     string `json:"host" validate:"required,max=255"`
	Port     int    `json:"port" validate:"omitempty,min=1,max=65535"`
	Username string `json:"username" validate:"required,max=200"`
	Password string `json:"password" validate:"max=200"`
	Rows     int    `json:"rows" validate:"omitempty,min=1,max=1000"`
	Cols     int    `json:"cols" validate:"omitempty,min=1,max=1000"`
}

type terminalResize struct {
	Rows int `json:"rows" validate:"required,min=1,max=1000"`
	Cols int `json:"cols" validate:"required,min=1,max=1000"`
}

type terminalView struct {
	ID    string    `json:"id"`
	Rows  int       `json:"rows"`
	Cols  int       `json:"cols"`
	Since time.Time `json:"since"`
}

func (terminal *Term) view() terminalView {
	terminal.mu.RLock()
	defer terminal.mu.RUnlock()
	return terminalView{ID: terminal.Id, Rows: terminal.Rows, Cols: terminal.Cols, Since: terminal.Since}
}

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/terminals", listTerminals, httpapi.Operation{ID: "listTerminals", Summary: "List in-memory SSH terminal sessions", Response: []terminalView{}})
	router.Handle(http.MethodPost, "/terminals", CreateTermHandler, httpapi.Operation{ID: "createTerminal", Summary: "Create an SSH session; default port/rows/cols are 22/24/80; uses existing host-key verification", Status: http.StatusCreated, Request: terminalCreate{}, Response: terminalView{}})
	router.Handle(http.MethodGet, "/terminals/:id", getTerminal, httpapi.Operation{ID: "getTerminal", Summary: "Read terminal metadata", Response: terminalView{}})
	router.Handle(http.MethodPatch, "/terminals/:id", SetTermWindowSizeHandler, httpapi.Operation{ID: "resizeTerminal", Summary: "Set terminal rows and cols; never mutates state through GET", Request: terminalResize{}, Response: terminalView{}})
	router.Handle(http.MethodDelete, "/terminals/:id", deleteTerminal, httpapi.Operation{ID: "deleteTerminal", Summary: "Close an SSH terminal session", Response: map[string]string{}})
	router.Handle(http.MethodGet, "/terminals/:id/data", LinkTermDataHandler, httpapi.Operation{ID: "connectTerminal", Summary: "Upgrade to WebSocket for interactive terminal I/O; not a JSON command execution API", Status: http.StatusSwitchingProtocols, ResponseMedia: "application/octet-stream", Response: ""})
}

func listTerminals(ctx *echo.Context) error {
	termMutex.Lock()
	defer termMutex.Unlock()
	terminals := make([]terminalView, 0, len(activeTerms))
	for _, terminal := range activeTerms {
		terminals = append(terminals, terminal.view())
	}
	sort.Slice(terminals, func(first, second int) bool { return terminals[first].ID < terminals[second].ID })
	return httpapi.OK(ctx, terminals)
}

func getTerminal(ctx *echo.Context) error {
	termMutex.Lock()
	terminal, exists := activeTerms[ctx.Param("id")]
	termMutex.Unlock()
	if !exists {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Terminal not found")
	}
	return httpapi.OK(ctx, terminal.view())
}

func deleteTerminal(ctx *echo.Context) error {
	identifier := ctx.Param("id")
	termMutex.Lock()
	terminal, exists := activeTerms[identifier]
	if exists {
		delete(activeTerms, identifier)
	}
	termMutex.Unlock()
	if !exists {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Terminal not found")
	}
	terminal.Close()
	return httpapi.OK(ctx, map[string]string{"id": identifier})
}
