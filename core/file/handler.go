package file

import (
	"fmt"
	"github.com/labstack/echo/v4"
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
)

func Process(c echo.Context) error {
	path := filepath.Clean(c.QueryParam("path"))
	mode := c.QueryParam("mode")
	switch c.Request().Method {
	case "GET":
		if mode == "edit" {
			file, err := os.Stat(path)
			if err != nil {
				return err
			}
			if file.Size() > 2*1024*1024 { //文件大于2M则不打开
				return echo.NewHTTPError(400, fmt.Sprintf("file too big %d", file.Size()))
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !utf8.Valid(data) { //这个检测不出来不知道为啥
				return echo.NewHTTPError(400, "not valid UTF-8")
			}
			return c.JSON(200, map[string]any{
				"type": filepath.Ext(file.Name()),
				"data": string(data),
			})
		}
		return c.Attachment(path, filepath.Base(path))
	case "POST":
		// 解析 Multipart 表单, `32<<20` 是最大内存限制
		err := c.Request().ParseMultipartForm(32 << 20)
		if err != nil {
			return err
		}
		files := c.Request().MultipartForm.File["files"] // 注意这里是 "files", 对应前端 input 的 name
		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				return err
			}
			destFile, err := os.Create(filepath.Join(path, fileHeader.Filename))
			if err != nil {
				return err
			}
			if _, err = io.Copy(destFile, file); err != nil {
				return err
			}
			destFile.Close()
			file.Close()
		}
		return c.JSON(200, "success")
	case "PUT":
		data, err := io.ReadAll(c.Request().Body)
		defer c.Request().Body.Close()
		if err != nil {
			return err
		}
		if mode == "rename" {
			if err = os.Rename(path, string(data)); err != nil {
				return err
			}
		}
		if mode == "PMSN" {
			perm, err := strconv.ParseUint(string(data), 8, 64)
			if err != nil {
				return err
			}
			if err = os.Chmod(path, os.FileMode(perm)); err != nil {
				return err
			}
		}
		if mode == "update" {

			if err = os.WriteFile(path, data, 0644); err != nil {
				return err
			}
		}
		if mode == "createFile" {
			if file, err := os.Create(path); err != nil && file.Close() == nil {
				return err
			}
		}
		if mode == "createFolder" {
			if err := os.MkdirAll(path, 0755); err != nil {
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
func Index(c echo.Context) error {
	//从cookie中获取目录路径
	directory := "/"
	if dirHistory, err := c.Cookie("dirHistory"); err == nil {
		if directory, err = url.QueryUnescape(dirHistory.Value); err != nil {
			return err
		}
	}

	if strings.HasPrefix(directory, "//") {
		directory = directory[1:]
	}
	//打开目录
	dir, err := os.Open(directory)
	if err != nil {
		c.SetCookie(&http.Cookie{Name: "dirHistory", Expires: time.Now(), MaxAge: -1})
		return err
	}
	defer dir.Close()

	//读取目录下的所有文件
	files, err := dir.Readdir(-1)
	if err != nil {
		c.SetCookie(&http.Cookie{Name: "dirHistory", Expires: time.Now(), MaxAge: -1})
		return err
	} else { //没有错误就排序一下，让文件夹在前面
		sort.Slice(files, func(i, j int) bool {
			return files[i].IsDir()
		})
	}

	//for _, file := range files {
	//	file.ModTime().Format("2006-01-02 15:04:05")
	//	fmt.Println(file.Mode().Perm())
	//	fmt.Println(file.Sys())
	//}
	c.SetCookie(&http.Cookie{Name: "dirHistory", Value: directory, Expires: time.Now(), MaxAge: 86400})
	return c.Render(http.StatusOK, "file.template", files)
}
