package file

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/labstack/echo/v5"
)

func Process(c *echo.Context) error {
	path := filepath.Clean(c.QueryParam("path"))
	mode := c.QueryParam("mode")
	switch c.Request().Method {
	case "GET":
		if mode == "edit" {
			file, err := os.Stat(path)
			if err != nil {
				return err
			}
			if file.Size() > 2*1024*1024 {
				return echo.NewHTTPError(400, fmt.Sprintf("file too big %d", file.Size()))
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !utf8.Valid(data) {
				return echo.NewHTTPError(400, "not valid UTF-8")
			}
			return c.JSON(200, map[string]any{
				"type": filepath.Ext(file.Name()),
				"data": string(data),
			})
		}
		return c.Attachment(path, filepath.Base(path))
	case "POST":
		err := c.Request().ParseMultipartForm(32 << 20)
		if err != nil {
			return err
		}
		files := c.Request().MultipartForm.File["files"]
		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				return err
			}
			destFile, err := os.Create(filepath.Join(path, fileHeader.Filename))
			if err != nil {
				file.Close()
				return err
			}
			_, err = io.Copy(destFile, file)
			destFile.Close()
			file.Close()
			if err != nil {
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
		switch mode {
		case "rename":
			if err = os.Rename(path, string(data)); err != nil {
				return err
			}
		case "PMSN":
			perm, err := strconv.ParseUint(string(data), 8, 32)
			if err != nil {
				return err
			}
			if err = os.Chmod(path, os.FileMode(perm)); err != nil {
				return err
			}
		case "update":
			if err = os.WriteFile(path, data, 0644); err != nil {
				return err
			}
		case "createFile":
			file, err := os.Create(path)
			if err != nil {
				return err
			}
			file.Close()
		case "createFolder":
			if err = os.MkdirAll(path, 0755); err != nil {
				return err
			}
		}
		return c.JSON(200, "success")
	case "DELETE":
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		return c.JSON(200, "success")
	}
	return echo.ErrMethodNotAllowed
}

// Index 主要依靠cookie来进行路径状态管理
func Index(c *echo.Context) error {
	directory := "/"
	if dirHistory, err := c.Cookie("dirHistory"); err == nil {
		if directory, err = url.QueryUnescape(dirHistory.Value); err != nil {
			return err
		}
	}

	if strings.HasPrefix(directory, "//") {
		directory = directory[1:]
	}

	dir, err := os.Open(directory)
	if err != nil {
		c.SetCookie(&http.Cookie{Name: "dirHistory", Expires: time.Now(), MaxAge: -1})
		return err
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		c.SetCookie(&http.Cookie{Name: "dirHistory", Expires: time.Now(), MaxAge: -1})
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].IsDir()
	})

	c.SetCookie(&http.Cookie{Name: "dirHistory", Value: url.QueryEscape(directory), Expires: time.Now(), MaxAge: 86400})
	return c.Render(http.StatusOK, "file.template", map[string]any{
		"Files": files,
		"Dir":   directory,
	})
}
