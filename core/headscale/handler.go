//go:build linux || darwin || freebsd

package headscale

import (
	"errors"
	_ "github.com/juanfont/headscale/cmd/headscale/cli"
	"github.com/juanfont/headscale/hscontrol"
	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
)

const confPath = "/etc/headscale/config.yaml"

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "POST":
		if c.QueryParam("status") == "start" {
			cfg, err := types.LoadServerConfig()
			if err != nil {
				return err
			}
			app, err := hscontrol.NewHeadscale(cfg)
			if err != nil {
				return err
			}
			if err = app.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}

		return c.JSON(200, "success")
	case "PUT":
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(confPath, data, 0644); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "GET":
		file, err := os.ReadFile(confPath)
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "headscale.template", map[string]any{
			"headscaleConfig": string(file),
			"headscaleEnable": viper.Get("enable.frps").(bool),
		})
	}

	return echo.ErrMethodNotAllowed
}
