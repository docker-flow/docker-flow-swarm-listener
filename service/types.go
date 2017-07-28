package service

import "github.com/docker/docker/api/types/swarm"

type SwarmService struct {
	swarm.Service
}
