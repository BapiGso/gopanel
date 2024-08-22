package monitor

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func Index(c echo.Context) error {
	if c.QueryParam("type") == "info" {
		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().WriteHeader(http.StatusOK)
		for {
			select {
			case <-c.Request().Context().Done():
				return nil
			default:
				jsonStu, err := json.Marshal(M)
				if err != nil {
					return err
				}
				fmt.Fprint(c.Response(), "data: "+string(jsonStu)+"\n\n")
				c.Response().Flush()
				time.Sleep(2 * time.Second)
				go M.refresh()
			}
		}
	}
	return c.Render(http.StatusOK, "monitor.template", nil)
}
