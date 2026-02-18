package firewall

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

var mgr Manager

func init() {
	m, err := NewManager()
	if err == nil {
		mgr = m
	}
}

func Index(c *echo.Context) error {
	if mgr == nil {
		return c.Render(200, "unavailable.template", nil)
	}

	switch c.Request().Method {
	case http.MethodGet:
		rules, err := mgr.List()
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "firewall.template", rules)

	case http.MethodPost:
		req := new(struct {
			Network   string `form:"network"`
			Protocol  string `form:"protocol"`
			Port      uint   `form:"port"`
			Direction string `form:"direction"`
			Action    string `form:"action"`
		})
		if err := c.Bind(req); err != nil {
			return err
		}
		rule := Rule{
			Network:   req.Network,
			Protocol:  req.Protocol,
			Port:      uint16(req.Port),
			Direction: req.Direction,
			Action:    req.Action,
		}
		if err := mgr.Add(rule); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "success"})

	case http.MethodDelete:
		id := c.QueryParam("id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing id parameter"})
		}
		if err := mgr.Delete(id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "success"})
	}

	return echo.ErrMethodNotAllowed
}
