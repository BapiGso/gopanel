package docker

import (
	"github.com/docker/docker/client"
)

var dockerCLI = func() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	return cli
}
