package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "GET":
		if c.QueryParam("type") == "info" {
			c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
			c.Response().WriteHeader(http.StatusOK)
			for {
				select {
				case <-c.Request().Context().Done():
					return nil
				default:
					images, err := apiClient.ImageList(context.Background(), image.ListOptions{All: true})
					if err != nil {
						return err
					}
					containers, err := apiClient.ContainerList(context.Background(), container.ListOptions{All: true})
					if err != nil {
						return err
					}

					jsonStu, err := json.Marshal(map[string]any{
						"images":     images,
						"containers": containers,
					})
					if err != nil {
						return err
					}
					fmt.Fprint(c.Response(), "data: "+string(jsonStu)+"\n\n")
					c.Response().Flush()
					time.Sleep(2 * time.Second)
				}
			}
		}
		if apiClientErr != nil {
			return apiClientErr
		}
		return c.Render(http.StatusOK, "docker.template", nil)
	case "PUT":
		switch c.QueryParam("type") {
		case "pause":
			if err := apiClient.ContainerPause(context.Background(), c.QueryParam("id")); err != nil {
				return err
			}
		case "unpause":
			if err := apiClient.ContainerUnpause(context.Background(), c.QueryParam("id")); err != nil {
				return err
			}
		case "stop":
			if err := apiClient.ContainerStop(context.Background(), c.QueryParam("id"), container.StopOptions{}); err != nil {
				return err
			}
		case "restart":
			if err := apiClient.ContainerRestart(context.Background(), c.QueryParam("id"), container.StopOptions{}); err != nil {
				return err
			}
		case "remove":
			if err := apiClient.ContainerRemove(context.Background(), c.QueryParam("id"), container.RemoveOptions{}); err != nil {
				return err
			}
		case "ImageRemove":
			if remove, err := apiClient.ImageRemove(context.Background(), c.QueryParam("id"), image.RemoveOptions{}); err != nil {
				return err
			} else {
				return c.JSON(200, remove)
			}
		}
		return c.JSON(200, "success")
	}

	return echo.ErrMethodNotAllowed
}
