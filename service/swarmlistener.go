package service

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"../metrics"
)

// SwarmListening provides public api for interacting with swarm listener
type SwarmListening interface {
	Run()
	NotifyServices(ignoreCache bool)
	NotifyNodes(ignoreCache bool)
	GetServicesParameters(ctx context.Context) ([]map[string]string, error)
	GetNodesParameters(ctx context.Context) ([]map[string]string, error)
}

// SwarmListener provides public api
type SwarmListener struct {
	SSListener         SwarmServiceListening
	SSClient           SwarmServiceInspector
	SSCache            SwarmServiceCacher
	SSEventChan        chan Event
	SSNotificationChan chan Notification

	NodeListener         NodeListening
	NodeClient           NodeInspector
	NodeCache            NodeCacher
	NodeEventChan        chan Event
	NodeNotificationChan chan Notification

	NotifyDistributor NotifyDistributing

	ServiceCreateCancelManager CancelManaging
	ServiceRemoveCancelManager CancelManaging
	IncludeNodeInfo            bool
	IgnoreKey                  string
	IncludeKey                 string
	Log                        *log.Logger
}

func newSwarmListener(
	ssListener SwarmServiceListening,
	ssClient SwarmServiceInspector,
	ssCache SwarmServiceCacher,

	nodeListener NodeListening,
	nodeClient NodeInspector,
	nodeCache NodeCacher,

	notifyDistributor NotifyDistributing,

	serviceCreateCancelManager CancelManaging,
	serviceRemoveCancelManager CancelManaging,
	includeNodeInfo bool,
	ignoreKey string,
	includeKey string,
	logger *log.Logger,
) *SwarmListener {

	return &SwarmListener{
		SSListener:                 ssListener,
		SSClient:                   ssClient,
		SSCache:                    ssCache,
		SSEventChan:                make(chan Event),
		SSNotificationChan:         make(chan Notification),
		NodeListener:               nodeListener,
		NodeClient:                 nodeClient,
		NodeCache:                  nodeCache,
		NodeEventChan:              make(chan Event),
		NodeNotificationChan:       make(chan Notification),
		NotifyDistributor:          notifyDistributor,
		ServiceCreateCancelManager: serviceCreateCancelManager,
		ServiceRemoveCancelManager: serviceRemoveCancelManager,
		IncludeNodeInfo:            includeNodeInfo,
		IgnoreKey:                  ignoreKey,
		IncludeKey:                 includeKey,
		Log:                        logger,
	}
}

// NewSwarmListenerFromEnv creats `SwarmListener` from environment variables
func NewSwarmListenerFromEnv(retries, interval int, logger *log.Logger) (*SwarmListener, error) {
	ignoreKey := os.Getenv("DF_NOTIFY_LABEL")
	includeNodeInfo := os.Getenv("DF_INCLUDE_NODE_IP_INFO") == "true"

	dockerClient, err := NewDockerClientFromEnv()
	if err != nil {
		return nil, err
	}
	ssListener := NewSwarmServiceListener(dockerClient, logger)
	ssClient := NewSwarmServiceClient(dockerClient, ignoreKey, "com.df.scrapeNetwork", logger)
	ssCache := NewSwarmServiceCache()

	nodeListener := NewNodeListener(dockerClient, logger)
	nodeClient := NewNodeClient(dockerClient)
	nodeCache := NewNodeCache()

	notifyDistributor := NewNotifyDistributorFromEnv(retries, interval, logger)

	return newSwarmListener(
		ssListener,
		ssClient,
		ssCache,
		nodeListener,
		nodeClient,
		nodeCache,
		notifyDistributor,
		NewCancelManager(1, false),
		NewCancelManager(1, false),
		includeNodeInfo,
		ignoreKey,
		"com.docker.stack.namespace",
		logger,
	), nil

}

// Run starts swarm listener
func (l *SwarmListener) Run() {
	l.connectServiceChannels()
	l.connectNodeChannels()

	if l.SSEventChan != nil {
		l.SSListener.ListenForServiceEvents(l.SSEventChan)
	}
	if l.NodeEventChan != nil {
		l.NodeListener.ListenForNodeEvents(l.NodeEventChan)
	}

	l.NotifyDistributor.Run(l.SSNotificationChan, l.NodeNotificationChan)
}

func (l *SwarmListener) connectServiceChannels() {

	// Remove service channels if there are no service listeners
	if !l.NotifyDistributor.HasServiceListeners() {
		l.SSEventChan = nil
		l.SSNotificationChan = nil
		return
	}

	go func() {
		for event := range l.SSEventChan {
			if event.Type == EventTypeCreate {
				go l.processServiceEventCreate(event)
			} else {
				go l.processServiceEventRemove(event)
			}
		}
	}()
}

type internalParams struct {
	ID     string
	Params string
}

func (l *SwarmListener) processServiceEventCreate(event Event) {
	l.ServiceRemoveCancelManager.ForceDelete(event.ID)
	ctx := l.ServiceCreateCancelManager.Add(event.ID, event.TimeNano)
	defer l.ServiceCreateCancelManager.Delete(event.ID, event.TimeNano)

	paramsChan := make(chan internalParams)

	go func() {
		service, err := l.SSClient.SwarmServiceInspect(ctx, event.ID, l.IncludeNodeInfo)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				return
			}
			l.Log.Printf("ERROR: %v", err)
			return
		}
		// Ignored service (filtered by `com.df.notify`)
		if service == nil {
			return
		}
		ssm := MinifySwarmService(*service, l.IgnoreKey, l.IncludeKey)

		// Store in cache
		isUpdated := l.SSCache.InsertAndCheck(ssm)
		if !isUpdated {
			return
		}
		metrics.RecordService(l.SSCache.Len())

		params := GetSwarmServiceMiniCreateParameters(ssm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		paramsChan <- internalParams{
			ID: ssm.ID, Params: paramsEncoded,
		}
	}()

	for {
		select {
		case params := <-paramsChan:
			l.placeOnNotificationChan(l.SSNotificationChan, event.Type, event.TimeNano, params.ID, params.Params)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (l *SwarmListener) processServiceEventRemove(event Event) {
	l.ServiceCreateCancelManager.ForceDelete(event.ID)
	ctx := l.ServiceRemoveCancelManager.Add(event.ID, event.TimeNano)
	defer l.ServiceRemoveCancelManager.Delete(event.ID, event.TimeNano)

	paramsChan := make(chan internalParams)
	go func() {
		ssm, ok := l.SSCache.Get(event.ID)
		if !ok {
			return
		}
		l.SSCache.Delete(ssm.ID)
		metrics.RecordService(l.SSCache.Len())

		params := GetSwarmServiceMiniRemoveParameters(ssm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		paramsChan <- internalParams{
			ID: ssm.ID, Params: paramsEncoded,
		}
	}()

	for {
		select {
		case params := <-paramsChan:
			l.placeOnNotificationChan(l.SSNotificationChan, event.Type, event.TimeNano, params.ID, params.Params)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (l *SwarmListener) connectNodeChannels() {

	// Remove node channels if there are no service listeners
	if !l.NotifyDistributor.HasNodeListeners() {
		l.NodeEventChan = nil
		l.NodeNotificationChan = nil
		return
	}

	go func() {
		for event := range l.NodeEventChan {
			if event.Type == EventTypeCreate {
				node, err := l.NodeClient.NodeInspect(event.ID)
				if err != nil {
					l.Log.Printf("ERROR: %v", err)
					continue
				}
				nm := MinifyNode(node)

				// Store in cache
				isUpdated := l.NodeCache.InsertAndCheck(nm)
				if !isUpdated {
					continue
				}
				params := GetNodeMiniCreateParameters(nm)
				paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
				l.placeOnNotificationChan(l.NodeNotificationChan, event.Type, event.TimeNano, nm.ID, paramsEncoded)
			} else {
				// EventTypeRemove
				nm, ok := l.NodeCache.Get(event.ID)
				if !ok {
					continue
				}
				l.NodeCache.Delete(nm.ID)

				params := GetNodeMiniRemoveParameters(nm)
				paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
				l.placeOnNotificationChan(l.NodeNotificationChan, event.Type, event.TimeNano, nm.ID, paramsEncoded)
			}
		}
	}()
}

// NotifyServices places all services on queue to notify services on service events
func (l SwarmListener) NotifyServices(useCache bool) {
	services, err := l.SSClient.SwarmServiceList(context.Background(), l.IncludeNodeInfo)
	if err != nil {
		l.Log.Printf("ERROR: NotifyService, %v", err)
		return
	}

	nowTimeNano := time.Now().UTC().UnixNano()
	if useCache {
		// Send to event chan, which uses the cache
		go func() {
			for _, s := range services {
				l.placeOnEventChan(l.SSEventChan, EventTypeCreate, s.ID, nowTimeNano)
			}
		}()
	} else {
		// Send directly to notification chan, skipping the cache
		go func() {
			for _, s := range services {
				ssm := MinifySwarmService(s, l.IgnoreKey, l.IncludeKey)

				params := GetSwarmServiceMiniCreateParameters(ssm)
				paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
				l.placeOnNotificationChan(l.SSNotificationChan, EventTypeCreate, nowTimeNano, ssm.ID, paramsEncoded)
			}
		}()
	}
}

// NotifyNodes places all services on queue to notify serivces on node events
func (l SwarmListener) NotifyNodes(useCache bool) {
	nodes, err := l.NodeClient.NodeList(context.Background())
	if err != nil {
		l.Log.Printf("ERROR: NotifyNodes, %v", err)
		return
	}

	nowTimeNano := time.Now().UTC().UnixNano()
	if useCache {
		// Send to event chan, which uses the cache
		go func() {
			for _, n := range nodes {
				l.placeOnEventChan(l.NodeEventChan, EventTypeCreate, n.ID, nowTimeNano)
			}
		}()
	} else {
		// Send directly to notification chan, skiping the cache
		go func() {
			for _, n := range nodes {
				nm := MinifyNode(n)
				params := GetNodeMiniCreateParameters(nm)
				paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
				l.placeOnNotificationChan(l.NodeNotificationChan, EventTypeCreate, nowTimeNano, nm.ID, paramsEncoded)
			}
		}()
	}
}

func (l SwarmListener) placeOnNotificationChan(notiChan chan<- Notification, eventType EventType, timeNano int64, ID string, parameters string) {
	notiChan <- Notification{
		EventType:  eventType,
		ID:         ID,
		Parameters: parameters,
		TimeNano:   timeNano,
	}
}

func (l SwarmListener) placeOnEventChan(eventChan chan<- Event, eventType EventType, ID string, timeNano int64) {
	eventChan <- Event{
		Type:     eventType,
		ID:       ID,
		TimeNano: timeNano,
	}
}

// GetServicesParameters get all services
func (l SwarmListener) GetServicesParameters(ctx context.Context) ([]map[string]string, error) {
	services, err := l.SSClient.SwarmServiceList(ctx, l.IncludeNodeInfo)
	if err != nil {
		return []map[string]string{}, err
	}
	params := []map[string]string{}
	for _, s := range services {
		ssm := MinifySwarmService(s, l.IgnoreKey, l.IncludeKey)
		newParams := GetSwarmServiceMiniCreateParameters(ssm)
		if len(newParams) > 0 {
			params = append(params, newParams)
		}
	}
	return params, nil
}

// GetNodesParameters get all nodes
func (l SwarmListener) GetNodesParameters(ctx context.Context) ([]map[string]string, error) {
	nodes, err := l.NodeClient.NodeList(ctx)
	if err != nil {
		return []map[string]string{}, err
	}
	params := []map[string]string{}
	for _, n := range nodes {
		mn := MinifyNode(n)
		newParams := GetNodeMiniCreateParameters(mn)
		if len(newParams) > 0 {
			params = append(params, newParams)
		}
	}
	return params, nil
}
