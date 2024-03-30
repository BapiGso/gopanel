package website

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path/filepath"
)

func Index(c echo.Context) error {

	dir := "websiteConfig"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}
	var caddyFile []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".caddy" {
			caddyFile = append(caddyFile, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "website.template", map[string]any{
		"caddyFile": caddyFile,
	})
}
