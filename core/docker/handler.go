package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/labstack/echo/v5"
	"net/http"
	"time"
)

func Index(c *echo.Context) error {
	switch c.Request().Method {
	case "GET":
		if c.QueryParam("type") == "info" {
			return streamDockerInfo(c)
		}
		return c.Render(http.StatusOK, "docker.template", nil)
	case "PUT":
		if apiClientErr != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": apiClientErr.Error()})
		}
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

func streamDockerInfo(c *echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := writeDockerInfo(c); err != nil {
			return err
		}

		select {
		case <-c.Request().Context().Done():
			return nil
		case <-ticker.C:
		}
	}
}

func writeDockerInfo(c *echo.Context) error {
	payload := map[string]any{
		"images":     []any{},
		"containers": []any{},
	}

	if apiClientErr != nil {
		payload["error"] = apiClientErr.Error()
	} else {
		images, err := apiClient.ImageList(c.Request().Context(), image.ListOptions{All: true})
		if err != nil {
			payload["error"] = err.Error()
		} else {
			containers, err := apiClient.ContainerList(c.Request().Context(), container.ListOptions{All: true})
			if err != nil {
				payload["error"] = err.Error()
			} else {
				payload["images"] = images
				payload["containers"] = containers
			}
		}
	}

	jsonStu, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprint(c.Response(), "data: "+string(jsonStu)+"\n\n"); err != nil {
		return err
	}
	return http.NewResponseController(c.Response()).Flush()
}
