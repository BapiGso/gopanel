package monitor

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func Index(c echo.Context) error {
	return c.Render(http.StatusOK, "monitor.template", nil)
}

func StreamInfo(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().WriteHeader(http.StatusOK)
	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			jsonStu, err := json.Marshal(M)
			if err != nil {
				fmt.Println("生成json字符串错误")
			}
			fmt.Fprint(c.Response(), "data: "+string(jsonStu)+"\n\n")
			c.Response().Flush()
			time.Sleep(2 * time.Second)
			go M.refresh()
		}
	}
}
