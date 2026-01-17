package monitor

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v5"
	"net/http"
	"time"
)

// Index 是监视器页面和 SSE 流的 HTTP 处理器
func Index(c *echo.Context) error {
	if c.QueryParam("type") == "info" { // SSE 流请求
		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
		c.Response().Header().Set(echo.HeaderConnection, "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		for {
			select {
			case <-c.Request().Context().Done(): // 客户端断开连接
				return nil
			default:
				// 直接序列化全局的 SysMonitor 实例。
				// 注意：这里没有锁保护，SysMonitor 可能在序列化过程中被 fetchAllStats 修改，
				// 从而产生竞态条件。
				SysMonitor.fetchAllStats()
				jsonStu, err := json.Marshal(SysMonitor)
				if err != nil {
					return err
				}

				if _, err := fmt.Fprint(c.Response(), "data: "+string(jsonStu)+"\n\n"); err != nil {
					return err
				}
				http.NewResponseController(c.Response()).Flush()
				time.Sleep(2 * time.Second)
			}
		}
	}
	return c.Render(http.StatusOK, "monitor.template", nil)
}
