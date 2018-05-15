package service

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
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

// CreateRemoveCancelManager combines two cancel managers for creating and
// removing services
type CreateRemoveCancelManager struct {
	createCancelManager CancelManaging
	removeCancelManager CancelManaging
	mux                 sync.RWMutex
}

// AddEvent controls canceling for creating and removing services
// A create event will cancel delete events with the same ID
// A remove event will cancel create events with the same ID
func (c *CreateRemoveCancelManager) AddEvent(event Event) context.Context {
	c.mux.Lock()
	defer c.mux.Unlock()
	if event.Type == EventTypeCreate {
		c.removeCancelManager.ForceDelete(event.ID)
		return c.createCancelManager.Add(context.Background(), event.ID, event.TimeNano)
	}
	// EventTypeRemove
	c.createCancelManager.ForceDelete(event.ID)
	return c.removeCancelManager.Add(context.Background(), event.ID, event.TimeNano)
}

// RemoveEvent removes and cancels event from its corresponding
// cancel manager
func (c *CreateRemoveCancelManager) RemoveEvent(event Event) bool {
	c.mux.Lock()
	defer c.mux.Unlock()
	if event.Type == EventTypeCreate {
		return c.createCancelManager.Delete(event.ID, event.TimeNano)
	}
	// EventTypeRemove
	return c.removeCancelManager.Delete(event.ID, event.TimeNano)
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

	ServiceCreateRemoveCancelManager *CreateRemoveCancelManager
	NodeCreateRemoveCancelManager    *CreateRemoveCancelManager
	IncludeNodeInfo                  bool
	IgnoreKey                        string
	IncludeKey                       string
	Log                              *log.Logger
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
	nodeCreateCancelManager CancelManaging,
	nodeRemoveCancelManager CancelManaging,
	includeNodeInfo bool,
	ignoreKey string,
	includeKey string,
	logger *log.Logger,
) *SwarmListener {

	return &SwarmListener{
		SSListener:           ssListener,
		SSClient:             ssClient,
		SSCache:              ssCache,
		SSEventChan:          make(chan Event),
		SSNotificationChan:   make(chan Notification),
		NodeListener:         nodeListener,
		NodeClient:           nodeClient,
		NodeCache:            nodeCache,
		NodeEventChan:        make(chan Event),
		NodeNotificationChan: make(chan Notification),
		NotifyDistributor:    notifyDistributor,
		ServiceCreateRemoveCancelManager: &CreateRemoveCancelManager{
			createCancelManager: serviceCreateCancelManager,
			removeCancelManager: serviceRemoveCancelManager},
		NodeCreateRemoveCancelManager: &CreateRemoveCancelManager{
			createCancelManager: nodeCreateCancelManager,
			removeCancelManager: nodeRemoveCancelManager},
		IncludeNodeInfo: includeNodeInfo,
		IgnoreKey:       ignoreKey,
		IncludeKey:      includeKey,
		Log:             logger,
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
		NewCancelManager(false),
		NewCancelManager(false),
		NewCancelManager(false),
		NewCancelManager(false),
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

func (l *SwarmListener) processServiceEventCreate(event Event) {
	ctx := l.ServiceCreateRemoveCancelManager.AddEvent(event)
	defer l.ServiceCreateRemoveCancelManager.RemoveEvent(event)

	doneChan := make(chan struct{})

	go func() {
		service, err := l.SSClient.SwarmServiceInspect(ctx, event.ID, l.IncludeNodeInfo)
		if err != nil {
			if !strings.Contains(err.Error(), "context canceled") {
				l.Log.Printf("ERROR: %v", err)
			}
			return
		}
		// Ignored service (filtered by `com.df.notify`)
		if service == nil {
			return
		}
		ssm := MinifySwarmService(*service, l.IgnoreKey, l.IncludeKey)

		if event.UseCache {
			// Store in cache
			isUpdated := l.SSCache.InsertAndCheck(ssm)
			if !isUpdated {
				return
			}
			metrics.RecordService(l.SSCache.Len())
		}

		params := GetSwarmServiceMiniCreateParameters(ssm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(
			l.SSNotificationChan, event.Type, event.TimeNano, ssm.ID, paramsEncoded, doneChan)
	}()

	for {
		select {
		case <-doneChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (l *SwarmListener) processServiceEventRemove(event Event) {
	ctx := l.ServiceCreateRemoveCancelManager.AddEvent(event)
	defer l.ServiceCreateRemoveCancelManager.RemoveEvent(event)

	doneChan := make(chan struct{})

	go func() {

		ssm, ok := l.SSCache.Get(event.ID)
		if !ok {
			return
		}
		l.SSCache.Delete(ssm.ID)
		metrics.RecordService(l.SSCache.Len())

		params := GetSwarmServiceMiniRemoveParameters(ssm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(
			l.SSNotificationChan, event.Type, event.TimeNano, ssm.ID, paramsEncoded, doneChan)
	}()

	for {
		select {
		case <-doneChan:
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
				go l.processNodeEventCreate(event)
			} else {
				go l.processNodeEventRemove(event)
			}
		}
	}()
}

func (l *SwarmListener) processNodeEventCreate(event Event) {
	ctx := l.NodeCreateRemoveCancelManager.AddEvent(event)
	defer l.NodeCreateRemoveCancelManager.RemoveEvent(event)

	doneChan := make(chan struct{})

	go func() {

		node, err := l.NodeClient.NodeInspect(event.ID)
		if err != nil {
			if !strings.Contains(err.Error(), "context canceled") {
				l.Log.Printf("ERROR: %v", err)
			}
			return
		}
		nm := MinifyNode(node)

		if event.UseCache {
			// Store in cache
			isUpdated := l.NodeCache.InsertAndCheck(nm)
			if !isUpdated {
				return
			}
		}
		params := GetNodeMiniCreateParameters(nm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(l.NodeNotificationChan, event.Type, event.TimeNano, nm.ID, paramsEncoded, doneChan)
	}()

	for {
		select {
		case <-doneChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (l *SwarmListener) processNodeEventRemove(event Event) {
	ctx := l.NodeCreateRemoveCancelManager.AddEvent(event)
	defer l.NodeCreateRemoveCancelManager.RemoveEvent(event)

	doneChan := make(chan struct{})
	go func() {
		nm, ok := l.NodeCache.Get(event.ID)
		if !ok {
			return
		}
		l.NodeCache.Delete(nm.ID)

		params := GetNodeMiniRemoveParameters(nm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(l.NodeNotificationChan, event.Type, event.TimeNano, nm.ID, paramsEncoded, doneChan)
	}()

	for {
		select {
		case <-doneChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// NotifyServices places all services on queue to notify services on service events
func (l SwarmListener) NotifyServices(useCache bool) {
	services, err := l.SSClient.SwarmServiceList(context.Background())
	if err != nil {
		l.Log.Printf("ERROR: NotifyService, %v", err)
		return
	}

	nowTimeNano := time.Now().UTC().UnixNano()
	go func() {
		for _, s := range services {
			l.placeOnEventChan(l.SSEventChan, EventTypeCreate, s.ID, nowTimeNano, useCache)
		}
	}()
}

// NotifyNodes places all services on queue to notify serivces on node events
func (l SwarmListener) NotifyNodes(useCache bool) {
	nodes, err := l.NodeClient.NodeList(context.Background())
	if err != nil {
		l.Log.Printf("ERROR: NotifyNodes, %v", err)
		return
	}

	nowTimeNano := time.Now().UTC().UnixNano()
	go func() {
		for _, n := range nodes {
			l.placeOnEventChan(l.NodeEventChan, EventTypeCreate, n.ID, nowTimeNano, useCache)
		}
	}()
}

func (l SwarmListener) placeOnNotificationChan(notiChan chan<- Notification, eventType EventType, timeNano int64, ID string, parameters string, doneChan chan struct{}) {
	notiChan <- Notification{
		EventType:  eventType,
		ID:         ID,
		Parameters: parameters,
		TimeNano:   timeNano,
		Done:       doneChan,
	}
}

func (l SwarmListener) placeOnEventChan(eventChan chan<- Event, eventType EventType, ID string, timeNano int64, useCache bool) {
	eventChan <- Event{
		Type:     eventType,
		ID:       ID,
		TimeNano: timeNano,
		UseCache: useCache,
	}
}

// GetServicesParameters get all services
func (l SwarmListener) GetServicesParameters(ctx context.Context) ([]map[string]string, error) {
	params := []map[string]string{}

	services, err := l.SSClient.SwarmServiceList(ctx)
	if err != nil {
		return params, err
	}

	// concurrent
	var wg sync.WaitGroup
	paramsChan := make(chan map[string]string)
	done := make(chan struct{})

	for _, ss := range services {
		wg.Add(1)
		go func(ss SwarmService) {
			defer wg.Done()
			if l.IncludeNodeInfo {
				if nodeInfo, err := l.SSClient.GetNodeInfo(ctx, ss, true); err == nil {
					ss.NodeInfo = nodeInfo
				}
			}
			ssm := MinifySwarmService(ss, l.IgnoreKey, l.IncludeKey)
			newParams := GetSwarmServiceMiniCreateParameters(ssm)
			if len(newParams) > 0 {
				paramsChan <- newParams
			}
		}(ss)
	}

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

L:
	for {
		select {
		case p := <-paramsChan:
			params = append(params, p)
		case <-done:
			break L
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
