package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// SwarmServiceInspector is able to inspect services
type SwarmServiceInspector interface {
	SwarmServiceInspect(ctx context.Context, serviceID string, includeNodeIPInfo bool) (*SwarmService, error)
	SwarmServiceList(ctx context.Context) ([]SwarmService, error)
	GetNodeInfo(ctx context.Context, ss SwarmService) (NodeIPSet, error)
	SwarmServiceRunning(ctx context.Context, serviceID string) (bool, error)
}

// SwarmServiceClient implements `SwarmServiceInspector` for docker
type SwarmServiceClient struct {
	DockerClient      *client.Client
	FilterLabel       string
	FilterKey         string
	ScrapeNetLabel    string
	ServiceNamePrefix string
	Log               *log.Logger
}

// NewSwarmServiceClient creates a `SwarmServiceClient`
func NewSwarmServiceClient(
	c *client.Client, filterLabel, scrapNetLabel string, serviceNamePrefix string, logger *log.Logger) *SwarmServiceClient {
	key := strings.SplitN(filterLabel, "=", 2)[0]
	return &SwarmServiceClient{DockerClient: c,
		FilterLabel:       filterLabel,
		FilterKey:         key,
		ScrapeNetLabel:    scrapNetLabel,
		ServiceNamePrefix: serviceNamePrefix,
		Log:               logger,
	}
}

// SwarmServiceInspect returns `SwarmService` from its ID
// Returns nil when service doesnt not have the `FilterLabel`
// When `includeNodeIPInfo` is true, return node info as well
func (c SwarmServiceClient) SwarmServiceInspect(ctx context.Context, serviceID string, includeNodeIPInfo bool) (*SwarmService, error) {
	service, _, err := c.DockerClient.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return nil, err
	}

	// Check if service has label
	if _, ok := service.Spec.Labels[c.FilterKey]; !ok {
		return nil, nil
	}

	if len(c.ServiceNamePrefix) > 0 {
		service.Spec.Name = fmt.Sprintf("%s_%s", c.ServiceNamePrefix, service.Spec.Name)
	}

	ss := SwarmService{service, nil}

	// Always wait for service to converge
	taskList, err := GetTaskList(ctx, c.DockerClient, ss.ID)
	if err != nil {
		return nil, err
	}
	if includeNodeIPInfo {
		if nodeInfo, err := c.getNodeInfo(ctx, taskList, service); err == nil {
			ss.NodeInfo = nodeInfo
		}
	}
	return &ss, nil
}

// SwarmServiceList returns a list of services
func (c SwarmServiceClient) SwarmServiceList(ctx context.Context) ([]SwarmService, error) {
	filter := filters.NewArgs()
	filter.Add("label", c.FilterLabel)
	services, err := c.DockerClient.ServiceList(ctx, types.ServiceListOptions{Filters: filter})
	if err != nil {
		return nil, err
	}
	swarmServices := []SwarmService{}
	for _, s := range services {
		if len(c.ServiceNamePrefix) > 0 {
			s.Spec.Name = fmt.Sprintf("%s_%s", c.ServiceNamePrefix, s.Spec.Name)
		}
		ss := SwarmService{s, nil}
		swarmServices = append(swarmServices, ss)
	}
	return swarmServices, nil
}

// GetNodeInfo returns node info for swarm service
func (c SwarmServiceClient) GetNodeInfo(ctx context.Context, ss SwarmService) (NodeIPSet, error) {

	// For services that do not have `ScrapeNetLabel` will
	// early exit, and avoid getting the task list
	_, ok := ss.Spec.Labels[c.ScrapeNetLabel]
	if !ok {
		return nil, nil
	}

	taskList, err := GetTaskList(ctx, c.DockerClient, ss.ID)
	if err != nil {
		return NodeIPSet{}, err
	}
	return c.getNodeInfo(ctx, taskList, ss.Service)
}

// SwarmServiceRunning returns true if service is running
func (c SwarmServiceClient) SwarmServiceRunning(ctx context.Context, serviceID string) (bool, error) {
	return TasksAllRunning(ctx, c.DockerClient, serviceID)
}

func (c SwarmServiceClient) getNodeInfo(ctx context.Context, taskList []swarm.Task, ss swarm.Service) (NodeIPSet, error) {

	networkName, ok := ss.Spec.Labels[c.ScrapeNetLabel]
	if !ok {
		return nil, fmt.Errorf("Unable to get NodeInfo: %s label is not defined for service %s", c.ScrapeNetLabel, ss.Spec.Name)
	}

	nodeInfo := NodeIPSet{}
	nodeIPCache := map[string]string{}
	for _, task := range taskList {
		if len(task.NetworksAttachments) == 0 || len(task.NetworksAttachments[0].Addresses) == 0 {
			continue
		}
		var address string
		for _, networkAttach := range task.NetworksAttachments {
			if networkAttach.Network.Spec.Name == networkName && len(networkAttach.Addresses) > 0 {
				address = strings.Split(networkAttach.Addresses[0], "/")[0]
			}
		}

		if len(address) == 0 {
			continue
		}

		if nodeName, ok := nodeIPCache[task.NodeID]; ok {
			nodeInfo.Add(nodeName, address, task.NodeID)
		} else {
			node, _, err := c.DockerClient.NodeInspectWithRaw(ctx, task.NodeID)
			if err != nil {
				continue
			}
			nodeInfo.Add(node.Description.Hostname, address, task.NodeID)
			nodeIPCache[task.NodeID] = node.Description.Hostname
		}
	}

	if nodeInfo.Cardinality() == 0 {
		return nil, nil
	}
	return nodeInfo, nil
}
