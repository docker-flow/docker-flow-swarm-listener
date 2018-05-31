package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
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
	SSListener SwarmServiceListening
	SSClient   SwarmServiceInspector
	SSCache    SwarmServiceCacher
	SSPoller   SwarmServicePolling

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
	UseDockerServiceEvents           bool
	IgnoreKey                        string
	IncludeKey                       string
	Log                              *log.Logger
}

func newSwarmListener(
	ssListener SwarmServiceListening,
	ssClient SwarmServiceInspector,
	ssCache SwarmServiceCacher,
	ssPoller SwarmServicePolling,
	ssEventChan chan Event,
	ssNotificationChan chan Notification,

	nodeListener NodeListening,
	nodeClient NodeInspector,
	nodeCache NodeCacher,
	nodeEventChan chan Event,
	nodeNotificationChan chan Notification,

	notifyDistributor NotifyDistributing,

	serviceCreateCancelManager CancelManaging,
	serviceRemoveCancelManager CancelManaging,
	nodeCreateCancelManager CancelManaging,
	nodeRemoveCancelManager CancelManaging,
	includeNodeInfo bool,
	useDockerServiceEvents bool,
	ignoreKey string,
	includeKey string,
	logger *log.Logger,
) *SwarmListener {

	return &SwarmListener{
		SSListener:           ssListener,
		SSClient:             ssClient,
		SSCache:              ssCache,
		SSPoller:             ssPoller,
		SSEventChan:          ssEventChan,
		SSNotificationChan:   ssNotificationChan,
		NodeListener:         nodeListener,
		NodeClient:           nodeClient,
		NodeCache:            nodeCache,
		NodeEventChan:        nodeEventChan,
		NodeNotificationChan: nodeNotificationChan,
		NotifyDistributor:    notifyDistributor,
		ServiceCreateRemoveCancelManager: &CreateRemoveCancelManager{
			createCancelManager: serviceCreateCancelManager,
			removeCancelManager: serviceRemoveCancelManager},
		NodeCreateRemoveCancelManager: &CreateRemoveCancelManager{
			createCancelManager: nodeCreateCancelManager,
			removeCancelManager: nodeRemoveCancelManager},
		IncludeNodeInfo:        includeNodeInfo,
		UseDockerServiceEvents: useDockerServiceEvents,
		IgnoreKey:              ignoreKey,
		IncludeKey:             includeKey,
		Log:                    logger,
	}
}

// NewSwarmListenerFromEnv creats `SwarmListener` from environment variables
func NewSwarmListenerFromEnv(
	retries, interval, servicePollingInterval int, logger *log.Logger) (*SwarmListener, error) {
	ignoreKey := os.Getenv("DF_NOTIFY_LABEL")
	includeNodeInfo, err := strconv.ParseBool(os.Getenv("DF_INCLUDE_NODE_IP_INFO"))
	if err != nil {
		includeNodeInfo = false
	}
	useDockerServiceEvents, err := strconv.ParseBool(os.Getenv("DF_USE_DOCKER_SERVICE_EVENTS"))
	if err != nil {
		useDockerServiceEvents = false
	}

	dockerClient, err := NewDockerClientFromEnv()
	if err != nil {
		return nil, err
	}
	notifyDistributor := NewNotifyDistributorFromEnv(retries, interval, logger)

	var ssListener *SwarmServiceListener
	var ssClient *SwarmServiceClient
	var ssCache *SwarmServiceCache
	var ssEventChan chan Event
	var ssNotificationChan chan Notification

	var nodeListener *NodeListener
	var nodeClient *NodeClient
	var nodeCache *NodeCache
	var nodeEventChan chan Event
	var nodeNotificationChan chan Notification

	if notifyDistributor.HasServiceListeners() {
		ssListener = NewSwarmServiceListener(dockerClient, logger)
		ssClient = NewSwarmServiceClient(dockerClient, ignoreKey, "com.df.scrapeNetwork", logger)
		ssCache = NewSwarmServiceCache()
		ssEventChan = make(chan Event)
		ssNotificationChan = make(chan Notification)
	}

	if notifyDistributor.HasNodeListeners() {
		nodeListener = NewNodeListener(dockerClient, logger)
		nodeClient = NewNodeClient(dockerClient)
		nodeCache = NewNodeCache()
		nodeEventChan = make(chan Event)
		nodeNotificationChan = make(chan Notification)
	}

	ssPoller := NewSwarmServicePoller(
		ssClient, ssCache, servicePollingInterval, includeNodeInfo,
		func(ss SwarmService) SwarmServiceMini {
			return MinifySwarmService(ss, ignoreKey, "com.docker.stack.namespace")
		}, logger)

	return newSwarmListener(
		ssListener,
		ssClient,
		ssCache,
		ssPoller,
		ssEventChan,
		ssNotificationChan,
		nodeListener,
		nodeClient,
		nodeCache,
		nodeEventChan,
		nodeNotificationChan,
		notifyDistributor,
		NewCancelManager(true),
		NewCancelManager(true),
		NewCancelManager(true),
		NewCancelManager(true),
		includeNodeInfo,
		useDockerServiceEvents,
		ignoreKey,
		"com.docker.stack.namespace",
		logger,
	), nil

}

// Run starts swarm listener
func (l *SwarmListener) Run() {

	if l.NotifyDistributor.HasServiceListeners() {
		l.connectServiceChannels()

		if l.UseDockerServiceEvents {
			l.SSListener.ListenForServiceEvents(l.SSEventChan)
			l.Log.Printf("Listening to Docker Service Events")
		}

		go l.SSPoller.Run(l.SSEventChan)
	}
	if l.NotifyDistributor.HasNodeListeners() {
		l.connectNodeChannels()
		l.NodeListener.ListenForNodeEvents(l.NodeEventChan)
		l.Log.Printf("Listening to Docker Node Events")
	}

	l.NotifyDistributor.Run(l.SSNotificationChan, l.NodeNotificationChan)
}

func (l *SwarmListener) connectServiceChannels() {

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

	errChan := make(chan error)

	go func() {
		service, err := l.SSClient.SwarmServiceInspect(ctx, event.ID, l.IncludeNodeInfo)
		if err != nil {
			errChan <- err
			return
		}
		// Ignored service (filtered by `com.df.notify`)
		if service == nil {
			errChan <- nil
			return
		}
		ssm := MinifySwarmService(*service, l.IgnoreKey, l.IncludeKey)

		if event.UseCache {
			// Store in cache
			isUpdated := l.SSCache.InsertAndCheck(ssm)
			if !isUpdated {
				errChan <- nil
				return
			}
			metrics.RecordService(l.SSCache.Len())
		}

		params := GetSwarmServiceMiniCreateParameters(ssm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(
			l.SSNotificationChan, event.Type, event.TimeNano, ssm.ID, paramsEncoded, errChan)
	}()

	for {
		select {
		case err := <-errChan:
			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					l.Log.Printf("ERROR: %v", err)
				}
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func (l *SwarmListener) processServiceEventRemove(event Event) {
	ctx := l.ServiceCreateRemoveCancelManager.AddEvent(event)
	defer l.ServiceCreateRemoveCancelManager.RemoveEvent(event)

	errChan := make(chan error)

	go func() {

		ssm, ok := l.SSCache.Get(event.ID)
		if !ok {
			errChan <- fmt.Errorf("%s not in cache", event.ID)
			return
		}
		params := GetSwarmServiceMiniRemoveParameters(ssm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(
			l.SSNotificationChan, event.Type, event.TimeNano, ssm.ID, paramsEncoded, errChan)
	}()

	for {
		select {
		case err := <-errChan:
			if err != nil {
				if !strings.Contains(err.Error(), "not in cache") {
					l.Log.Printf("ERROR: %v", err)
				}
				return
			}
			l.SSCache.Delete(event.ID)
			metrics.RecordService(l.SSCache.Len())
			return
		case <-ctx.Done():
			return
		}
	}
}

func (l *SwarmListener) connectNodeChannels() {

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

	errChan := make(chan error)

	go func() {

		node, err := l.NodeClient.NodeInspect(event.ID)
		if err != nil {
			errChan <- err
			return
		}
		nm := MinifyNode(node)

		if event.UseCache {
			// Store in cache
			isUpdated := l.NodeCache.InsertAndCheck(nm)
			if !isUpdated {
				errChan <- nil
				return
			}
		}
		params := GetNodeMiniCreateParameters(nm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(l.NodeNotificationChan, event.Type, event.TimeNano, nm.ID, paramsEncoded, errChan)
	}()

	for {
		select {
		case err := <-errChan:
			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					l.Log.Printf("ERROR: %v", err)
				}
				return
			}
			l.NotifyServices(true)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (l *SwarmListener) processNodeEventRemove(event Event) {
	ctx := l.NodeCreateRemoveCancelManager.AddEvent(event)
	defer l.NodeCreateRemoveCancelManager.RemoveEvent(event)

	errChan := make(chan error)
	go func() {
		nm, ok := l.NodeCache.Get(event.ID)
		if !ok {
			errChan <- fmt.Errorf("%s not in cache", event.ID)
			return
		}

		params := GetNodeMiniRemoveParameters(nm)
		paramsEncoded := ConvertMapStringStringToURLValues(params).Encode()
		l.placeOnNotificationChan(l.NodeNotificationChan, event.Type, event.TimeNano, nm.ID, paramsEncoded, errChan)
	}()

	for {
		select {
		case err := <-errChan:
			if err != nil {
				if !strings.Contains(err.Error(), "not in cache") {
					l.Log.Printf("ERROR: %v", err)
				}
				return
			}
			l.NodeCache.Delete(event.ID)
			l.NotifyServices(true)
			return
		case <-ctx.Done():
			return
		}
	}
}

// NotifyServices places all services on queue to notify services on service events
func (l SwarmListener) NotifyServices(useCache bool) {

	if !l.NotifyDistributor.HasServiceListeners() {
		return
	}

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

	if !l.NotifyDistributor.HasNodeListeners() {
		return
	}

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

func (l SwarmListener) placeOnNotificationChan(notiChan chan<- Notification, eventType EventType, timeNano int64, ID string, parameters string, errorChan chan error) {
	notiChan <- Notification{
		EventType:  eventType,
		ID:         ID,
		Parameters: parameters,
		TimeNano:   timeNano,
		ErrorChan:  errorChan,
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
				if nodeInfo, err := l.SSClient.GetNodeInfo(ctx, ss); err == nil {
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
