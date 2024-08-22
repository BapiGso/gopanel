package docker

import (
	"github.com/docker/docker/client"
)

var apiClient, apiClientErr = func() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv)
}()
