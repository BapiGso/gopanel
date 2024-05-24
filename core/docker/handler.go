package docker

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo/v4"
	"net/http"
)

func Index(c echo.Context) error {
	switch c.Request().Method {
	case "GET":
		if c.QueryParam("type") == "info" {
			containers, err := apiClient.ContainerList(context.Background(), container.ListOptions{All: true})
			if err != nil {
				return err
			}
			return c.JSON(200, containers)
		}
		return c.Render(http.StatusOK, "docker.template", nil)
	}

	//// 设置容器配置
	//resp, err := apiClient.ContainerCreate(context.Background(), &container.Config{
	//	Image: "nginx",
	//	ExposedPorts: nat.PortSet{
	//		"80/tcp": struct{}{},
	//	},
	//}, nil, nil, nil, "")
	//if err != nil {
	//	return err
	//}
	//
	//// 启动容器
	//if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
	//	return err
	//}
	//
	//fmt.Println("Container started successfully")
	//
	//// 停止容器
	//if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{}); err != nil {
	//	return err
	//}
	return c.Render(http.StatusOK, "docker.template", nil)
}
