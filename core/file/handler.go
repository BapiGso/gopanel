package file

import (
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

type Handler struct {
	service *LocalFileService
}

var defaultHandler = mustNewHandler(filesystemRoot())

func NewHandler(basePath string) (*Handler, error) {
	service, err := NewLocalFileService(basePath)
	if err != nil {
		return nil, err
	}
	return &Handler{service: service}, nil
}

func mustNewHandler(basePath string) *Handler {
	handler, err := NewHandler(basePath)
	if err != nil {
		panic(err)
	}
	return handler
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
	listing := make([]fileEntry, 0, len(files))
	for _, entry := range files {
		listing = append(listing, metadata(path.Join(directory, entry.Name()), entry))
	}
	return c.Render(http.StatusOK, "file.template", map[string]any{
		"Files": listing,
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
