package service

import (
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// EventListener listens for docker service events
type EventListener struct {
	*client.Client
}

// Event contains information about docker service events
type Event struct {
	Action    string
	ServiceID string
}

// NewEventListener returns a new instance of the `EventListener` structure
func NewEventListener(host string) *EventListener {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	dc, err := client.NewClient(host, dockerApiVersion, nil, defaultHeaders)
	if err != nil {
		logPrintf(err.Error())
	}
	return &EventListener{dc}
}

// NewEventListenerFromEnv returns a new instance of the `EventListener` structure using environment variable `DF_DOCKER_HOST` for the host
func NewEventListenerFromEnv() *EventListener {
	host := "unix:///var/run/docker.sock"
	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	return NewEventListener(host)
}

// ListenForEvents returns a stream of Events
func (s *EventListener) ListenForEvents() (<-chan Event, <-chan error) {

	events := make(chan Event)
	errs := make(chan error, 1)
	started := make(chan struct{})

	go func() {
		defer close(errs)
		filter := filters.NewArgs()
		filter.Add("type", "service")
		eventStream, eventErrors := s.Events(
			context.Background(),
			types.EventsOptions{Filters: filter},
		)

		close(started)
		for {
			select {
			case msg := <-eventStream:
				events <- Event{
					Action:    msg.Action,
					ServiceID: msg.Actor.ID,
				}
			case err := <-eventErrors:
				logPrintf("%v", err)
				errs <- err
				return
			}
		}

	}()
	<-started

	return events, errs
}
