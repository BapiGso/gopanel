package file

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

func Download(c echo.Context) error {
	mode := c.QueryParam("mode")
	path := c.QueryParam("path")
	if strings.HasPrefix(path, "//") {
		path = path[1:]
	}
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
			"type": 1,
			"data": string(data),
		})
	}
	return c.File(path)
}

// Index 主要依靠cookie来进行路径状态管理
func Index(c echo.Context) error {
	//从cookie中获取目录路径
	directory := "/"
	if dirHistory, err := c.Cookie("dirHistory"); err == nil {
		directory = dirHistory.Value
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

	for _, file := range files {
		file.ModTime().Format("2006-01-02 15:04:05")
		//fmt.Println(file.Mode().Perm())
		//fmt.Println(file.Sys())
	}
	c.SetCookie(&http.Cookie{Name: "dirHistory", Value: directory, Expires: time.Now(), MaxAge: 86400})
	return c.Render(http.StatusOK, "file.template", files)
}

func DownloadHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "download.template", nil)
}
