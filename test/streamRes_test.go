package test

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"testing"
	"time"
)

func TestSreamRes(t *testing.T) {

	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().WriteHeader(http.StatusOK)

		for {
			select {
			case <-c.Request().Context().Done():
				return nil
			default:
			}
			fmt.Fprint(c.Response(), "data: hi\n\n")
			c.Response().Flush()
			time.Sleep(1 * time.Second)
		}
		return nil
	})
	e.Logger.Fatal(e.Start(":1323"))
}
