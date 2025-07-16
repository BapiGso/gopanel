package netbird

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/netbirdio/netbird/client/cmd"
	"github.com/netbirdio/netbird/client/proto"
	"net/http"
	"os"
)

// Index
func Index(c echo.Context) error {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	//client := proto.NewDaemonServiceClient(nil)
	//fmt.Println(client)
	return c.Render(http.StatusOK, "netbird.template", nil)
}
