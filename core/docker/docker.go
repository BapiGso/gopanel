package docker

import (
	"github.com/docker/docker/client"
	"log/slog"
)

var apiClient = func() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		slog.Error(err.Error())
	}
	return cli
}()
