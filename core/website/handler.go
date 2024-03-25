package website

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path/filepath"
)

func Index(c echo.Context) error {
	if _, err := os.Stat("./caddy"); os.IsNotExist(err) {
		fmt.Println("Caddy不存在，开始下载...")
		installCaddy()
	}
	dir := "websiteConfig"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0755)
		return err
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
