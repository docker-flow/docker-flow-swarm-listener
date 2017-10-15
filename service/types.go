package service

import "github.com/docker/docker/api/types/swarm"

// SwarmService defines internal structure with service information
type SwarmService struct {
	swarm.Service
}
