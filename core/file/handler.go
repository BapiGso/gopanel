package file

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
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

var defaultHandler = NewHandler(afero.NewOsFs(), os.Getenv("FILE_BASE_PATH"))

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
	requestPath := c.QueryParam("path")
	if requestPath == "" {
		requestPath = "/"
	}
	mode := c.QueryParam("mode")

	switch c.Request().Method {
	case http.MethodGet:
		if mode == "edit" {
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

	case http.MethodPost:
		if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
			return err
		}

		files := c.Request().MultipartForm.File["files"]
		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				return err
			}
			if err := h.service.SaveUploadedFile(requestPath, fileHeader.Filename, file); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
		return c.JSON(http.StatusOK, "success")

	case http.MethodPut:
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}

		switch mode {
		case "rename":
			err = h.service.Rename(requestPath, string(data))
		case "PMSN":
			err = h.service.Chmod(requestPath, string(data))
		case "update":
			err = h.service.WriteFile(requestPath, data)
		case "createFile":
			err = h.service.CreateFile(requestPath)
		case "createFolder":
			err = h.service.CreateFolder(requestPath)
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "invalid file operation")
		}
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, "success")

	case http.MethodDelete:
		if err := h.service.RemoveAll(requestPath); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, "success")
	}

	return echo.ErrMethodNotAllowed
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
