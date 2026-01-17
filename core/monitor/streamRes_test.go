package monitor

import (
	"fmt"
	"github.com/labstack/echo/v5"
	"net/http"
	"testing"
	"time"
)

func TestSreamRes(t *testing.T) {
	e := echo.New()
	e.GET("/", func(c *echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().WriteHeader(http.StatusOK)

		for {
			select {
			case <-c.Request().Context().Done():
				return nil
			default:
			}
			fmt.Fprint(c.Response(), "data: hi\n\n")
			http.NewResponseController(c.Response()).Flush()
			time.Sleep(1 * time.Second)
		}
		return nil
	})
	e.Logger.Error(e.Start(":1323").Error())
}
