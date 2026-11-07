package file

import (
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/spf13/afero"
)

type Handler struct {
	service *LocalFileService
}

var defaultHandler = NewHandler(afero.NewOsFs(), filesystemRoot())

func NewHandler(fs afero.Fs, basePath string) *Handler {
	return &Handler{
		service: NewLocalFileService(fs, basePath),
	}
}

func DefaultHandler() *Handler {
	return defaultHandler
}

func Index(c *echo.Context) error {
	return defaultHandler.Index(c)
}

// Index 主要依靠cookie来进行路径状态管理
func (h *Handler) Index(c *echo.Context) error {
	directory := "/"
	if dirHistory, err := c.Cookie("dirHistory"); err == nil {
		if directory, err = url.QueryUnescape(dirHistory.Value); err != nil {
			return err
		}
	}

	directory = normalizeLogicalPath(directory)
	if _, err := h.service.resolve(directory); err != nil {
		directory = "/"
	}

	files, err := h.service.ReadDir(directory)
	if err != nil {
		c.SetCookie(&http.Cookie{Name: "dirHistory", Expires: time.Now(), MaxAge: -1})
		directory = "/"
		files, err = h.service.ReadDir(directory)
		if err != nil {
			return err
		}
	}

	sort.SliceStable(files, func(i, j int) bool {
		if files[i].IsDir() != files[j].IsDir() {
			return files[i].IsDir()
		}
		return files[i].Name() < files[j].Name()
	})

	c.SetCookie(&http.Cookie{Name: "dirHistory", Value: url.QueryEscape(directory), Expires: time.Now(), MaxAge: 86400})
	return c.Render(http.StatusOK, "file.template", map[string]any{
		"Files": files,
		"Dir":   directory,
	})
}

func normalizeLogicalPath(p string) string {
	if p == "" {
		return "/"
	}
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}
