package file

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/spf13/afero"
)

type Handler struct {
	service *LocalFileService
}

const (
	modeEdit         = "edit"
	modeRename       = "rename"
	modeChmod        = "chmod"
	modeChmodLegacy  = "PMSN"
	modeUpdate       = "update"
	modeCreateFile   = "createFile"
	modeCreateFolder = "createFolder"
)

var defaultHandler = NewHandler(afero.NewOsFs(), filesystemRoot())

func NewHandler(fs afero.Fs, basePath string) *Handler {
	return &Handler{
		service: NewLocalFileService(fs, basePath),
	}
}

func DefaultHandler() *Handler {
	return defaultHandler
}

func Process(c *echo.Context) error {
	return defaultHandler.Process(c)
}

func Index(c *echo.Context) error {
	return defaultHandler.Index(c)
}

func (h *Handler) Process(c *echo.Context) error {
	requestPath := requestPath(c)
	mode := c.QueryParam("mode")

	switch c.Request().Method {
	case http.MethodGet:
		return h.get(c, requestPath, mode)
	case http.MethodPost:
		return h.upload(c, requestPath)
	case http.MethodPut:
		return h.mutate(c, requestPath, mode)
	case http.MethodDelete:
		if err := h.service.RemoveAll(requestPath); err != nil {
			return err
		}
		return success(c)
	}

	return echo.ErrMethodNotAllowed
}

func requestPath(c *echo.Context) string {
	requestPath := c.QueryParam("path")
	if requestPath == "" {
		return "/"
	}
	return requestPath
}

func (h *Handler) get(c *echo.Context, requestPath, mode string) error {
	if mode == modeEdit {
		content, err := h.service.ReadEditable(requestPath)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type": filepath.Ext(filepath.Base(requestPath)),
			"data": content,
		})
	}

	file, err := h.service.Open(requestPath)
	if err != nil {
		return err
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+filepath.Base(requestPath)+`"`)
	return c.Stream(http.StatusOK, mime.TypeByExtension(filepath.Ext(requestPath)), file)
}

func (h *Handler) upload(c *echo.Context, dir string) error {
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return err
	}

	for _, fileHeader := range c.Request().MultipartForm.File["files"] {
		file, err := fileHeader.Open()
		if err != nil {
			return err
		}
		if err := h.service.SaveUploadedFile(dir, fileHeader.Filename, file); err != nil {
			file.Close()
			return err
		}
		file.Close()
	}
	return success(c)
}

func (h *Handler) mutate(c *echo.Context, requestPath, mode string) error {
	data, err := io.ReadAll(c.Request().Body)
	defer c.Request().Body.Close()
	if err != nil {
		return err
	}

	switch mode {
	case modeRename:
		err = h.service.Rename(requestPath, string(data))
	case modeChmod, modeChmodLegacy:
		err = h.service.Chmod(requestPath, string(data))
	case modeUpdate:
		err = h.service.WriteFile(requestPath, data)
	case modeCreateFile:
		err = h.service.CreateFile(requestPath)
	case modeCreateFolder:
		err = h.service.CreateFolder(requestPath)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file operation")
	}
	if err != nil {
		return err
	}
	return success(c)
}

func success(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
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
