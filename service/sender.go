package service

import (
	"github.com/docker/docker/api/types/swarm"
)

type Sender interface {
	ServicesCreate(services *[]swarm.Service, retries, interval int) error
	ServicesRemove(services *[]string, retries, interval int) error
}
