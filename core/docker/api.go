package docker

import (
	"context"
	"net/http"
	"time"

	"github.com/containerd/errdefs"
	"github.com/labstack/echo/v5"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"

	"gopanel/core/httpapi"
)

type snapshot struct {
	Images     []image.Summary     `json:"images"`
	Containers []container.Summary `json:"containers"`
}

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/docker/status", getStatus, httpapi.Operation{ID: "getDockerStatus", Summary: "Read one Docker snapshot without holding an SSE connection", Response: snapshot{}})
	router.Handle(http.MethodGet, "/docker/events", streamStatus, httpapi.Operation{ID: "streamDockerStatus", Summary: "Stream Docker snapshots as SSE JSON data/error envelopes", ResponseMedia: "text/event-stream", Response: ""})
	router.Handle(http.MethodGet, "/docker/containers", listContainers, httpapi.Operation{ID: "listDockerContainers", Summary: "List containers, including stopped containers", Response: []container.Summary{}})
	router.Handle(http.MethodGet, "/docker/images", listImages, httpapi.Operation{ID: "listDockerImages", Summary: "List local Docker images", Response: []image.Summary{}})
	for _, action := range []string{"start", "stop", "restart", "pause", "unpause"} {
		router.Handle(http.MethodPost, "/docker/containers/:id/"+action, containerAction(action), httpapi.Operation{ID: action + "DockerContainer", Summary: action + " a Docker container identified by id or name", Response: map[string]string{}})
	}
	router.Handle(http.MethodDelete, "/docker/containers/:id", containerAction("remove"), httpapi.Operation{ID: "deleteDockerContainer", Summary: "Remove a stopped container without forcing deletion", Response: map[string]string{}})
	router.Handle(http.MethodDelete, "/docker/images/:id", removeImage, httpapi.Operation{ID: "deleteDockerImage", Summary: "Remove a local image without forcing deletion", Response: map[string]string{}})
}

func dockerError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errdefs.IsNotFound(err):
		return httpapi.NewError(http.StatusNotFound, "not_found", err.Error())
	case errdefs.IsConflict(err):
		return httpapi.NewError(http.StatusConflict, "conflict", err.Error())
	case errdefs.IsInvalidArgument(err):
		return httpapi.NewError(http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		return httpapi.NewError(http.StatusServiceUnavailable, "docker_unavailable", err.Error())
	}
}

func collectSnapshot(ctx context.Context) (snapshot, error) {
	if apiClientErr != nil {
		return snapshot{}, dockerError(apiClientErr)
	}
	images, err := apiClient.ImageList(ctx, client.ImageListOptions{All: true})
	if err != nil {
		return snapshot{}, dockerError(err)
	}
	containers, err := apiClient.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return snapshot{}, dockerError(err)
	}
	result := snapshot{Images: images.Items, Containers: containers.Items}
	if result.Images == nil {
		result.Images = []image.Summary{}
	}
	if result.Containers == nil {
		result.Containers = []container.Summary{}
	}
	return result, nil
}

func getStatus(ctx *echo.Context) error {
	data, err := collectSnapshot(ctx.Request().Context())
	if err != nil {
		return err
	}
	return httpapi.OK(ctx, data)
}

func streamStatus(ctx *echo.Context) error {
	return httpapi.Stream(ctx, 2*time.Second, func() (any, error) { return collectSnapshot(ctx.Request().Context()) })
}

func listContainers(ctx *echo.Context) error {
	if apiClientErr != nil {
		return dockerError(apiClientErr)
	}
	result, err := apiClient.ContainerList(ctx.Request().Context(), client.ContainerListOptions{All: true})
	if err != nil {
		return dockerError(err)
	}
	if result.Items == nil {
		result.Items = []container.Summary{}
	}
	return httpapi.OK(ctx, result.Items)
}

func listImages(ctx *echo.Context) error {
	if apiClientErr != nil {
		return dockerError(apiClientErr)
	}
	result, err := apiClient.ImageList(ctx.Request().Context(), client.ImageListOptions{All: true})
	if err != nil {
		return dockerError(err)
	}
	if result.Items == nil {
		result.Items = []image.Summary{}
	}
	return httpapi.OK(ctx, result.Items)
}

func containerAction(action string) echo.HandlerFunc {
	return func(ctx *echo.Context) error {
		if apiClientErr != nil {
			return dockerError(apiClientErr)
		}
		containerID := httpapi.Param(ctx, "id")
		requestContext := ctx.Request().Context()
		var err error
		switch action {
		case "start":
			_, err = apiClient.ContainerStart(requestContext, containerID, client.ContainerStartOptions{})
		case "stop":
			_, err = apiClient.ContainerStop(requestContext, containerID, client.ContainerStopOptions{})
		case "restart":
			_, err = apiClient.ContainerRestart(requestContext, containerID, client.ContainerRestartOptions{})
		case "pause":
			_, err = apiClient.ContainerPause(requestContext, containerID, client.ContainerPauseOptions{})
		case "unpause":
			_, err = apiClient.ContainerUnpause(requestContext, containerID, client.ContainerUnpauseOptions{})
		case "remove":
			_, err = apiClient.ContainerRemove(requestContext, containerID, client.ContainerRemoveOptions{})
		}
		if err != nil {
			return dockerError(err)
		}
		return httpapi.OK(ctx, map[string]string{"id": containerID})
	}
}

func removeImage(ctx *echo.Context) error {
	if apiClientErr != nil {
		return dockerError(apiClientErr)
	}
	if _, err := apiClient.ImageRemove(ctx.Request().Context(), httpapi.Param(ctx, "id"), client.ImageRemoveOptions{}); err != nil {
		return dockerError(err)
	}
	return httpapi.OK(ctx, map[string]string{"id": httpapi.Param(ctx, "id")})
}
