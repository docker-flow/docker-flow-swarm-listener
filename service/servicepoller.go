package service

import (
	"context"
	"log"
	"time"
)

// SwarmServicePolling provides an interface for polling service changes
type SwarmServicePolling interface {
	Run(eventChan chan<- Event)
}

// SwarmServicePoller implements `SwarmServicePoller`
type SwarmServicePoller struct {
	SSClient        SwarmServiceInspector
	SSCache         SwarmServiceCacher
	PollingInterval int
	IncludeNodeInfo bool
	MinifyFunc      func(SwarmService) SwarmServiceMini
	Log             *log.Logger
}

// NewSwarmServicePoller creates a new `SwarmServicePoller`
func NewSwarmServicePoller(
	ssClient SwarmServiceInspector,
	ssCache SwarmServiceCacher,
	pollingInterval int,
	includeNodeInfo bool,
	minifyFunc func(SwarmService) SwarmServiceMini,
	log *log.Logger,
) *SwarmServicePoller {
	return &SwarmServicePoller{
		SSClient:        ssClient,
		SSCache:         ssCache,
		PollingInterval: pollingInterval,
		IncludeNodeInfo: includeNodeInfo,
		MinifyFunc:      minifyFunc,
		Log:             log,
	}
}

// Run starts poller and places events onto `eventChan`
func (s SwarmServicePoller) Run(
	eventChan chan<- Event) {

	if s.PollingInterval <= 0 {
		return
	}

	ctx := context.Background()

	s.Log.Printf("Polling for Service Changes")
	time.Sleep(time.Duration(s.PollingInterval) * time.Second)

	for {
		services, err := s.SSClient.SwarmServiceList(ctx)
		if err != nil {
			s.Log.Printf("ERROR (SwarmServicePolling): %v", err)
		} else {
			nowTimeNano := time.Now().UTC().UnixNano()
			keys := s.SSCache.Keys()
			for _, ss := range services {
				delete(keys, ss.ID)

				if s.IncludeNodeInfo {
					nodeInfo, err := s.SSClient.GetNodeInfo(ctx, ss)
					if err != nil {
						s.Log.Printf("ERROR: GetServicesParameters, %v", err)
					} else {
						ss.NodeInfo = nodeInfo
					}
				}

				ssMini := s.MinifyFunc(ss)
				if s.SSCache.IsNewOrUpdated(ssMini) {
					eventChan <- Event{
						Type:         EventTypeCreate,
						ID:           ss.ID,
						TimeNano:     nowTimeNano,
						ConsultCache: true,
					}
				}
			}

			// Remaining keys are removal events
			for k := range keys {
				eventChan <- Event{
					Type:         EventTypeRemove,
					ID:           k,
					TimeNano:     nowTimeNano,
					ConsultCache: true,
				}
			}
		}
		time.Sleep(time.Duration(s.PollingInterval) * time.Second)
	}
}
