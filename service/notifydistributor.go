package service

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

// Notification is a node notification
type Notification struct {
	EventType  EventType
	ID         string
	Parameters string
	TimeNano   int64
	Context    context.Context
	ErrorChan  chan error
}

type internalNotification struct {
	Notification
	Ctx context.Context
}

// NotifyEndpoint holds Notifiers and channels to watch
type NotifyEndpoint struct {
	ServiceNotifier NotificationSender
	NodeNotifier    NotificationSender
}

// NotifyDistributing takes a stream of `Notification` and
// NodeNotifiction and distributes it listeners
type NotifyDistributing interface {
	Run(serviceChan <-chan Notification, nodeChan <-chan Notification)
	HasServiceListeners() bool
	HasNodeListeners() bool
}

// NotifyDistributor distributes service and node notifications to `NotifyEndpoints`
// `NotifyEndpoints` are keyed by hostname to send notifications to
type NotifyDistributor struct {
	NotifyEndpoints      map[string]NotifyEndpoint
	ServiceCancelManager CancelManaging
	NodeCancelManager    CancelManaging
	log                  *log.Logger
	interval             int
}

func newNotifyDistributor(notifyEndpoints map[string]NotifyEndpoint,
	serviceCancelManager CancelManaging, nodeCancelManager CancelManaging,
	interval int, logger *log.Logger) *NotifyDistributor {
	return &NotifyDistributor{
		NotifyEndpoints:      notifyEndpoints,
		ServiceCancelManager: serviceCancelManager,
		NodeCancelManager:    nodeCancelManager,
		interval:             interval,
		log:                  logger,
	}
}

func newNotifyDistributorfromStrings(
	serviceCreateAddrs, serviceRemoveAddrs, nodeCreateAddrs, nodeRemoveAddrs,
	serviceCreateMethods, serviceRemoveMethods string,
	retries, interval int, logger *log.Logger) *NotifyDistributor {
	tempNotifyEP := map[string]map[string]string{}

	insertAddrStringIntoMap(
		tempNotifyEP, "createService", serviceCreateAddrs,
		"createServiceMethod", serviceCreateMethods)
	insertAddrStringIntoMap(
		tempNotifyEP, "removeService", serviceRemoveAddrs,
		"removeServiceMethod", serviceRemoveMethods)
	insertAddrStringIntoMap(
		tempNotifyEP, "createNode", nodeCreateAddrs,
		"createNodeMethod", http.MethodGet)
	insertAddrStringIntoMap(
		tempNotifyEP, "removeNode", nodeRemoveAddrs,
		"removeNodeMethod", http.MethodGet)

	notifyEndpoints := map[string]NotifyEndpoint{}

	for hostname, addrMap := range tempNotifyEP {
		ep := NotifyEndpoint{}
		if len(addrMap["createService"]) > 0 || len(addrMap["removeService"]) > 0 {
			ep.ServiceNotifier = NewNotifier(
				addrMap["createService"],
				addrMap["removeService"],
				addrMap["createServiceMethod"],
				addrMap["removeServiceMethod"],
				"service",
				retries,
				interval,
				logger,
			)
		}
		if len(addrMap["createNode"]) > 0 || len(addrMap["removeNode"]) > 0 {
			ep.NodeNotifier = NewNotifier(
				addrMap["createNode"],
				addrMap["removeNode"],
				addrMap["createNodeMethod"],
				addrMap["removeNodeMethod"],
				"node",
				retries,
				interval,
				logger,
			)
		}
		if ep.ServiceNotifier != nil || ep.NodeNotifier != nil {
			notifyEndpoints[hostname] = ep
		}
	}

	return newNotifyDistributor(
		notifyEndpoints,
		NewCancelManager(),
		NewCancelManager(),
		interval,
		logger)
}

func insertAddrStringIntoMap(
	tempEP map[string]map[string]string, key, addrs, methodsKey, methods string) {

	addrsList := strings.Split(addrs, ",")
	methodsList := strings.Split(methods, ",")
	maxMethodsIdx := len(methodsList) - 1

	for addrsIdx, v := range addrsList {
		urlObj, err := url.Parse(v)
		if err != nil {
			continue
		}
		host := urlObj.Host
		if len(host) == 0 {
			continue
		}
		if tempEP[host] == nil {
			tempEP[host] = map[string]string{}
		}
		tempEP[host][key] = v

		if addrsIdx <= maxMethodsIdx {
			tempEP[host][methodsKey] = methodsList[addrsIdx]
		} else {
			tempEP[host][methodsKey] = methodsList[maxMethodsIdx]
		}
	}
}

// NewNotifyDistributorFromEnv creates `NotifyDistributor` from environment variables
func NewNotifyDistributorFromEnv(retries, interval int, logger *log.Logger) *NotifyDistributor {
	var createServiceAddr, removeServiceAddr string
	if len(os.Getenv("DF_NOTIF_CREATE_SERVICE_URL")) > 0 {
		createServiceAddr = os.Getenv("DF_NOTIF_CREATE_SERVICE_URL")
	} else if len(os.Getenv("DF_NOTIFY_CREATE_SERVICE_URL")) > 0 {
		createServiceAddr = os.Getenv("DF_NOTIFY_CREATE_SERVICE_URL")
	} else {
		createServiceAddr = os.Getenv("DF_NOTIFICATION_URL")
	}
	if len(os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL")) > 0 {
		removeServiceAddr = os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL")
	} else if len(os.Getenv("DF_NOTIFY_REMOVE_SERVICE_URL")) > 0 {
		removeServiceAddr = os.Getenv("DF_NOTIFY_REMOVE_SERVICE_URL")
	} else {
		removeServiceAddr = os.Getenv("DF_NOTIFICATION_URL")
	}
	createNodeAddr := os.Getenv("DF_NOTIFY_CREATE_NODE_URL")
	removeNodeAddr := os.Getenv("DF_NOTIFY_REMOVE_NODE_URL")

	createServiceMethods := strings.ToUpper(os.Getenv("DF_NOTIFY_CREATE_SERVICE_METHOD"))
	removeServiceMethods := strings.ToUpper(os.Getenv("DF_NOTIFY_REMOVE_SERVICE_METHOD"))

	if len(createServiceMethods) == 0 {
		createServiceMethods = http.MethodGet
	}
	if len(removeServiceMethods) == 0 {
		removeServiceMethods = http.MethodGet
	}

	return newNotifyDistributorfromStrings(
		createServiceAddr, removeServiceAddr, createNodeAddr, removeNodeAddr,
		createServiceMethods, removeServiceMethods, retries, interval, logger)

}

// Run starts the distributor
func (d NotifyDistributor) Run(serviceChan <-chan Notification, nodeChan <-chan Notification) {

	if serviceChan != nil {
		go func() {
			for n := range serviceChan {
				go d.distributeServiceNotification(n)
			}
		}()
	}
	if nodeChan != nil {
		go func() {
			for n := range nodeChan {
				go d.distributeNodeNotification(n)
			}
		}()
	}
}

func (d NotifyDistributor) distributeServiceNotification(n Notification) {
	// Use time as request id
	ctx := d.ServiceCancelManager.Add(context.Background(), n.ID, n.TimeNano)
	defer d.ServiceCancelManager.Delete(n.ID, n.TimeNano)

	var wg sync.WaitGroup
	for _, endpoint := range d.NotifyEndpoints {
		wg.Add(1)
		go func(endpoint NotifyEndpoint) {
			defer wg.Done()
			d.processServiceNotification(ctx, n, endpoint)
		}(endpoint)
	}
	wg.Wait()

	if n.ErrorChan != nil {
		n.ErrorChan <- nil
	}
}

func (d NotifyDistributor) distributeNodeNotification(n Notification) {
	// Use time as request id
	ctx := d.NodeCancelManager.Add(context.Background(), n.ID, n.TimeNano)
	defer d.NodeCancelManager.Delete(n.ID, n.TimeNano)

	var wg sync.WaitGroup
	for _, endpoint := range d.NotifyEndpoints {
		wg.Add(1)
		go func(endpoint NotifyEndpoint) {
			defer wg.Done()
			d.processNodeNotification(ctx, n, endpoint)
		}(endpoint)
	}
	wg.Wait()
	if n.ErrorChan != nil {
		n.ErrorChan <- nil
	}
}

func (d NotifyDistributor) processServiceNotification(
	ctx context.Context, n Notification, endpoint NotifyEndpoint) {

	if endpoint.ServiceNotifier == nil {
		return
	}

	if n.EventType == EventTypeCreate {
		err := endpoint.ServiceNotifier.Create(ctx, n.Parameters)
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			d.log.Printf("ERROR: Unable to send ServiceCreateNotify to %s, params: %s", endpoint.ServiceNotifier.GetCreateAddr(), n.Parameters)
		}
	} else if n.EventType == EventTypeRemove {
		err := endpoint.ServiceNotifier.Remove(ctx, n.Parameters)
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			d.log.Printf("ERROR: Unable to send ServiceRemoveNotify to %s, params: %s", endpoint.ServiceNotifier.GetRemoveAddr(), n.Parameters)
		}
	}
}

func (d NotifyDistributor) processNodeNotification(
	ctx context.Context, n Notification, endpoint NotifyEndpoint) {

	if endpoint.NodeNotifier == nil {
		return
	}

	if n.EventType == EventTypeCreate {
		err := endpoint.NodeNotifier.Create(ctx, n.Parameters)
		if err != nil {
			d.log.Printf("ERROR: Unable to send NodeCreateNotify to %s, params: %s",
				endpoint.NodeNotifier.GetCreateAddr(), n.Parameters)
		}
	} else if n.EventType == EventTypeRemove {
		err := endpoint.NodeNotifier.Remove(ctx, n.Parameters)
		if err != nil {
			d.log.Printf("ERROR: Unable to send NodeRemoveNotify to %s, params: %s",
				endpoint.NodeNotifier.GetRemoveAddr(), n.Parameters)
		}
	}
}

// HasServiceListeners when there exists service listeners
func (d NotifyDistributor) HasServiceListeners() bool {
	for _, endpoint := range d.NotifyEndpoints {
		if endpoint.ServiceNotifier != nil {
			return true
		}
	}
	return false
}

// HasNodeListeners when there exists node listeners
func (d NotifyDistributor) HasNodeListeners() bool {
	for _, endpoint := range d.NotifyEndpoints {
		if endpoint.NodeNotifier != nil {
			return true
		}
	}
	return false
}
