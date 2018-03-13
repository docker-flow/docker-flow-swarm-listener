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
	SwarmServiceInspect(serviceID string, includeNodeIPInfo bool) (*SwarmService, error)
	SwarmServiceList(ctx context.Context, includeNodeIPInfo bool) ([]SwarmService, error)
}

// SwarmServiceClient implements `SwarmServiceInspector` for docker
type SwarmServiceClient struct {
	DockerClient   *client.Client
	FilterLabel    string
	FilterKey      string
	ScrapeNetLabel string
	Log            *log.Logger
}

// NewSwarmServiceClient creates a `SwarmServiceClient`
func NewSwarmServiceClient(c *client.Client, filterLabel, scrapNetLabel string, logger *log.Logger) *SwarmServiceClient {
	key := strings.SplitN(filterLabel, "=", 2)[0]
	return &SwarmServiceClient{DockerClient: c,
		FilterLabel:    filterLabel,
		FilterKey:      key,
		ScrapeNetLabel: scrapNetLabel,
		Log:            logger,
	}
}

// SwarmServiceInspect returns `SwarmService` from its ID
// Returns nil when service doesnt not have the `FilterLabel`
// When `includeNodeIPInfo` is true, return node info as well
func (c SwarmServiceClient) SwarmServiceInspect(serviceID string, includeNodeIPInfo bool) (*SwarmService, error) {
	service, _, err := c.DockerClient.ServiceInspectWithRaw(context.Background(), serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return nil, err
	}

	// Check if service has label
	if _, ok := service.Spec.Labels[c.FilterKey]; !ok {
		return nil, nil
	}

	ss := SwarmService{service, nil}
	if includeNodeIPInfo {
		nodeInfo, err := c.getNodeInfo(context.Background(), service)
		if err != nil {
			c.Log.Printf("%v", err)
		} else {
			ss.NodeInfo = nodeInfo
		}
	}
	return &ss, nil
}

// SwarmServiceList returns a list of services
// When `includeNodeIPInfo` is true, return node info as well
func (c SwarmServiceClient) SwarmServiceList(ctx context.Context, includeNodeIPInfo bool) ([]SwarmService, error) {
	filter := filters.NewArgs()
	filter.Add("label", c.FilterLabel)
	services, err := c.DockerClient.ServiceList(ctx, types.ServiceListOptions{Filters: filter})
	if err != nil {
		return nil, err
	}
	swarmServices := []SwarmService{}
	for _, s := range services {
		ss := SwarmService{s, nil}
		if includeNodeIPInfo {
			nodeInfo, _ := c.getNodeInfo(ctx, s)
			if err != nil {
				c.Log.Printf("%v", err)
			} else {
				ss.NodeInfo = nodeInfo
			}
		}
		swarmServices = append(swarmServices, ss)
	}
	return swarmServices, nil
}

func (c SwarmServiceClient) getNodeInfo(ctx context.Context, ss swarm.Service) (NodeIPSet, error) {

	networkName, ok := ss.Spec.Labels[c.ScrapeNetLabel]
	if !ok {
		return nil, fmt.Errorf("NodeInfo %s label is not defined for service %s", c.ScrapeNetLabel, ss.Spec.Name)
	}

	taskList, err := GetTaskList(ctx, c.DockerClient, ss.ID)
	if err != nil {
		return nil, err
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
			nodeInfo.Add(nodeName, address)
		} else {
			node, _, err := c.DockerClient.NodeInspectWithRaw(ctx, task.NodeID)
			if err != nil {
				continue
			}
			nodeInfo.Add(node.Description.Hostname, address)
			nodeIPCache[task.NodeID] = node.Description.Hostname
		}
	}

	if nodeInfo.Cardinality() == 0 {
		return nil, nil
	}
	return nodeInfo, nil
}
