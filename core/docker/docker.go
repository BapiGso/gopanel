package docker

import (
	"github.com/moby/moby/client"
)

var apiClient, apiClientErr = func() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}()
