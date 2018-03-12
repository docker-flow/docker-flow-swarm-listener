package service

import (
	"context"
	"log"
	"time"

	"../metrics"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// SwarmServiceListening listens for service events
type SwarmServiceListening interface {
	ListenForServiceEvents(chan<- Event)
}

// SwarmServiceListener listens for docker service events
type SwarmServiceListener struct {
	dockerClient *client.Client
	log          *log.Logger
}

// NewSwarmServiceListener creates a `SwarmServiceListener`
func NewSwarmServiceListener(c *client.Client, logger *log.Logger) *SwarmServiceListener {
	return &SwarmServiceListener{dockerClient: c, log: logger}
}

// ListenForServiceEvents listens for events and places them on channels
func (s SwarmServiceListener) ListenForServiceEvents(eventChan chan<- Event) {
	go func() {
		filter := filters.NewArgs()
		filter.Add("type", "service")
		msgStream, msgErrs := s.dockerClient.Events(
			context.Background(), types.EventsOptions{Filters: filter})

		for {
			select {
			case msg := <-msgStream:
				eventType := EventTypeCreate
				if msg.Action == "remove" {
					eventType = EventTypeRemove
				}
				eventChan <- Event{
					Type: eventType,
					ID:   msg.Actor.ID,
				}
			case err := <-msgErrs:
				s.log.Printf("%v, Restarting docker event stream", err)
				metrics.RecordError("ListenForServiceEvents")
				time.Sleep(time.Second)
				// Reopen event stream
				msgStream, msgErrs = s.dockerClient.Events(
					context.Background(), types.EventsOptions{Filters: filter})
			}
		}
	}()
}
