package file

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

// Handler 文件处理器
type Handler struct {
	service  FileService
	basePath string
}

// NewHandler 创建文件处理器
func NewHandler(basePath string) *Handler {
	return &Handler{
		service:  NewLocalFileService(basePath),
		basePath: basePath,
	}
}

// DefaultHandler 使用默认配置的处理函数
func DefaultHandler() *Handler {
	return NewHandler(GetDefaultBasePath())
}

// GetDefaultBasePath 获取默认基础路径
func GetDefaultBasePath() string {
	if bp := os.Getenv("FILE_BASE_PATH"); bp != "" {
		return filepath.Clean(bp)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// Index 文件管理首页
func (h *Handler) Index(c *echo.Context) error {
	directory := "/"
	if dirHistory, err := c.Cookie("dirHistory"); err == nil {
		if directory, err = url.QueryUnescape(dirHistory.Value); err != nil {
			return h.handleError(c, err)
		}
	}

	directory = strings.TrimPrefix(directory, "//")

	files, err := h.service.List(directory)
	if err != nil {
		c.SetCookie(&http.Cookie{Name: "dirHistory", Expires: time.Now(), MaxAge: -1})
		return h.handleError(c, err)
	}

	c.SetCookie(&http.Cookie{
		Name:    "dirHistory",
		Value:   url.QueryEscape(directory),
		Expires: time.Now().Add(24 * time.Hour),
		MaxAge:  86400,
	})

	return c.Render(http.StatusOK, "file.template", map[string]any{
		"Files": files,
		"Dir":   directory,
	})
}

// Process 文件操作处理（GET/POST/PUT/DELETE）
func (h *Handler) Process(c *echo.Context) error {
	path := c.QueryParam("path")
	mode := c.QueryParam("mode")

	switch c.Request().Method {
	case http.MethodGet:
		return h.handleGet(c, path, mode)
	case http.MethodPost:
		return h.handlePost(c, path)
	case http.MethodPut:
		return h.handlePut(c, path, mode)
	case http.MethodDelete:
		return h.handleDelete(c, path)
	default:
		return echo.ErrMethodNotAllowed
	}
}

// handleGet 处理 GET 请求（读取/下载）
func (h *Handler) handleGet(c *echo.Context, path, mode string) error {
	if mode == "edit" {
		content, err := h.service.Read(path)
		if err != nil {
			return h.handleError(c, err)
		}
		return c.JSON(200, content)
	}

	// 下载文件
	file, err := h.service.Download(path)
	if err != nil {
		return h.handleError(c, err)
	}
	defer file.Close()

	return c.Attachment(path, filepath.Base(path))
}

// handlePost 处理 POST 请求（上传）
func (h *Handler) handlePost(c *echo.Context, path string) error {
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return h.handleError(c, err)
	}

	files := c.Request().MultipartForm.File["files"]
	if len(files) == 0 {
		return h.jsonError(c, http.StatusBadRequest, "no files uploaded")
	}

	for _, fileHeader := range files {
		if err := h.uploadSingleFile(c, path, fileHeader); err != nil {
			return h.handleError(c, err)
		}
	}

	return c.JSON(200, map[string]string{"status": "success"})
}

// uploadSingleFile 上传单个文件
func (h *Handler) uploadSingleFile(c *echo.Context, dir string, fileHeader *multipart.FileHeader) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	targetPath := filepath.Join(dir, fileHeader.Filename)
	validatedPath, err := h.service.ValidatePath(targetPath)
	if err != nil {
		return err
	}

	dst, err := os.Create(validatedPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// handlePut 处理 PUT 请求（更新/创建/重命名/权限）
func (h *Handler) handlePut(c *echo.Context, path, mode string) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return h.handleError(c, err)
	}

	switch mode {
	case "rename":
		return h.handleRename(c, path, string(data))
	case "PMSN":
		return h.handleChmod(c, path, string(data))
	case "update":
		return h.handleUpdate(c, path, data)
	case "createFile":
		return h.handleCreate(c, path, false)
	case "createFolder":
		return h.handleCreate(c, path, true)
	default:
		return h.jsonError(c, http.StatusBadRequest, "invalid mode")
	}
}

// handleRename 处理重命名
func (h *Handler) handleRename(c *echo.Context, oldPath, newName string) error {
	// newName 是绝对路径格式，需要拼接目录
	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, filepath.Base(newName))

	if err := h.service.Rename(oldPath, newPath); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(200, map[string]string{"status": "success"})
}

// handleChmod 处理权限修改
func (h *Handler) handleChmod(c *echo.Context, path, perm string) error {
	if err := h.service.Chmod(path, perm); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(200, map[string]string{"status": "success"})
}

// handleUpdate 处理文件更新
func (h *Handler) handleUpdate(c *echo.Context, path string, data []byte) error {
	if err := h.service.Write(path, data); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(200, map[string]string{"status": "success"})
}

// handleCreate 处理创建文件/目录
func (h *Handler) handleCreate(c *echo.Context, path string, isDir bool) error {
	if err := h.service.Create(path, isDir); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(200, map[string]string{"status": "success"})
}

// handleDelete 处理 DELETE 请求
func (h *Handler) handleDelete(c *echo.Context, path string) error {
	if err := h.service.Delete(path); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(200, map[string]string{"status": "success"})
}

// handleError 统一错误处理
func (h *Handler) handleError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, ErrPathNotAllowed):
		return h.jsonError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrFileNotFound):
		return h.jsonError(c, http.StatusNotFound, "file not found")
	case errors.Is(err, ErrFileTooLarge):
		return h.jsonError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrInvalidUTF8):
		return h.jsonError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrInvalidPerm):
		return h.jsonError(c, http.StatusBadRequest, err.Error())
	default:
		// 检查是否是 HTTP 错误
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return err
		}
		return h.jsonError(c, http.StatusInternalServerError, err.Error())
	}
}

// jsonError 返回 JSON 格式错误
func (h *Handler) jsonError(c *echo.Context, code int, message string) error {
	return c.JSON(code, map[string]string{
		"error": message,
	})
}

// 兼容旧版本的导出函数

// Process 兼容旧版本的函数调用
func Process(c *echo.Context) error {
	return DefaultHandler().Process(c)
}

// Index 兼容旧版本的函数调用
func Index(c *echo.Context) error {
	return DefaultHandler().Index(c)
}
