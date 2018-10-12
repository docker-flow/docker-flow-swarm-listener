package service

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

// NodePolling provides an interface for polling node changes
type NodePolling interface {
	Run(eventChan chan<- Event)
}

// NodePoller implements `NodePolling`
type NodePoller struct {
	Client          NodeInspector
	Cache           NodeCacher
	PollingInterval int
	MinifyFunc      func(swarm.Node) NodeMini
	Log             *log.Logger
}

// NewNodePoller creates a new `NodePoller`
func NewNodePoller(
	client NodeInspector,
	cache NodeCacher,
	pollingInterval int,
	minifyFunc func(swarm.Node) NodeMini,
	log *log.Logger,
) *NodePoller {
	return &NodePoller{
		Client:          client,
		Cache:           cache,
		PollingInterval: pollingInterval,
		MinifyFunc:      minifyFunc,
		Log:             log,
	}
}

// Run starts poller and places events onto `eventChan`
func (n NodePoller) Run(eventChan chan<- Event) {

	if n.PollingInterval <= 0 {
		return
	}

	ctx := context.Background()

	n.Log.Printf("Polling for Node Changes")
	time.Sleep(time.Duration(n.PollingInterval) * time.Second)

	for {
		nodes, err := n.Client.NodeList(ctx)
		if err != nil {
			n.Log.Printf("ERROR (NodePoller): %v", err)
		} else {
			nowTimeNano := time.Now().UTC().UnixNano()
			keys := n.Cache.Keys()
			for _, node := range nodes {
				delete(keys, node.ID)

				nodeMini := n.MinifyFunc(node)
				if n.Cache.IsNewOrUpdated(nodeMini) {
					eventChan <- Event{
						Type:     EventTypeCreate,
						ID:       node.ID,
						TimeNano: nowTimeNano,
						ConsultCache: true,
					}
				}
			}

			// Remaining key sare removal events
			for k := range keys {
				eventChan <- Event{
					Type:     EventTypeRemove,
					ID:       k,
					TimeNano: nowTimeNano,
					ConsultCache: true,
				}
			}
		}
		time.Sleep(time.Duration(n.PollingInterval) * time.Second)
	}
}
