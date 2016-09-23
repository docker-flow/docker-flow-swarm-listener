package main

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type Service struct{}

var dockerClient = client.NewClient

func (m *Service) GetServices(host string) ([]swarm.Service, error) {
	dc, err := dockerClient(host, "", nil, nil)

	if err != nil {
		return []swarm.Service{}, err
	}

	services, err := dc.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		return []swarm.Service{}, err
	}

	return services, nil
}

func NewServices() Service {
	return Service{}
}
