package service

import (
	"os"

	"github.com/docker/docker/client"
)

var dockerAPIVersion = "v1.25"

// NewDockerClientFromEnv returns a `*client.Client` struct using environment variable
// `DF_DOCKER_HOST` for the host
func NewDockerClientFromEnv() (*client.Client, error) {
	host := "unix:///var/run/docker.sock"
	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	return client.NewClient(host, dockerAPIVersion, nil, defaultHeaders)
}
