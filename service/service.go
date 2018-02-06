package service

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// CachedServices stores the information about services processed by the system
var CachedServices map[string]SwarmService

// Service defines the based structure
type Service struct {
	Host                 string
	ServiceLastUpdatedAt time.Time
	DockerClient         *client.Client
}

// Servicer defines interface with mandatory methods
type Servicer interface {
	GetServices() (*[]SwarmService, error)
	GetNewServices(services *[]SwarmService) (*[]SwarmService, error)
	GetRemovedServices(services *[]SwarmService) *[]string
	GetServicesParameters(services *[]SwarmService) *[]map[string]string
}

// GetServicesParameters returns parameters extracted from labels associated with input services
func (m *Service) GetServicesParameters(services *[]SwarmService) *[]map[string]string {
	params := []map[string]string{}
	for _, s := range *services {
		sParams := getServiceParams(&s)
		if len(sParams) > 0 {
			params = append(params, sParams)
		}
	}
	return &params
}

// GetServices returns all services running in the cluster
func (m *Service) GetServices() (*[]SwarmService, error) {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("%s=true", os.Getenv("DF_NOTIFY_LABEL")))
	services, err := m.DockerClient.ServiceList(
		context.Background(),
		types.ServiceListOptions{Filters: filter},
	)
	if err != nil {
		logPrintf(err.Error())
		return &[]SwarmService{}, err
	}
	swarmServices := []SwarmService{}
	for _, s := range services {
		ss := SwarmService{s, nil}
		if strings.EqualFold(os.Getenv("DF_INCLUDE_NODE_IP_INFO"), "true") {
			ss.NodeInfo = m.getNodeInfo(ss)
		}
		swarmServices = append(swarmServices, ss)
	}
	return &swarmServices, nil
}

// GetNewServices returns services that were not processed previously
func (m *Service) GetNewServices(services *[]SwarmService) (*[]SwarmService, error) {
	newServices := []SwarmService{}
	tmpUpdatedAt := m.ServiceLastUpdatedAt
	for _, s := range *services {
		if tmpUpdatedAt.Nanosecond() == 0 || s.Meta.UpdatedAt.After(tmpUpdatedAt) {
			updated := false
			if service, ok := CachedServices[s.Spec.Name]; ok {
				if m.isUpdated(s, service) {
					updated = true
				}
			} else if !hasZeroReplicas(&s) {
				updated = true
			}
			if updated {
				newServices = append(newServices, s)
				CachedServices[s.Spec.Name] = s
				if m.ServiceLastUpdatedAt.Before(s.Meta.UpdatedAt) {
					m.ServiceLastUpdatedAt = s.Meta.UpdatedAt
				}
			}
		}
	}
	return &newServices, nil
}

// GetRemovedServices returns services that were removed from the cluster
func (m *Service) GetRemovedServices(services *[]SwarmService) *[]string {
	tmpMap := make(map[string]SwarmService)
	for k, v := range CachedServices {
		tmpMap[k] = v
	}
	for _, v := range *services {
		if _, ok := CachedServices[v.Spec.Name]; ok && !hasZeroReplicas(&v) {
			delete(tmpMap, v.Spec.Name)
		}
	}
	rs := []string{}
	for k := range tmpMap {
		rs = append(rs, k)
	}
	return &rs
}

// NewService returns a new instance of the `Service` structure
func NewService(host string) *Service {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	dc, err := client.NewClient(host, dockerApiVersion, nil, defaultHeaders)
	if err != nil {
		logPrintf(err.Error())
	}
	CachedServices = make(map[string]SwarmService)
	return &Service{
		Host:         host,
		DockerClient: dc,
	}
}

// NewServiceFromEnv returns a new instance of the `Service` structure using environment variable `DF_DOCKER_HOST` for the host
func NewServiceFromEnv() *Service {
	host := "unix:///var/run/docker.sock"
	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	return NewService(host)
}

func (m *Service) isUpdated(candidate SwarmService, cached SwarmService) bool {
	for k, v := range candidate.Spec.Labels {
		if strings.HasPrefix(k, "com.df.") {
			if storedValue, ok := cached.Spec.Labels[k]; !ok || v != storedValue {
				return true
			}
		}
	}
	if candidate.Service.Spec.Mode.Replicated != nil {
		candidateReplicas := candidate.Service.Spec.Mode.Replicated.Replicas
		cachedReplicas := cached.Service.Spec.Mode.Replicated.Replicas
		if *candidateReplicas > 0 && *candidateReplicas != *cachedReplicas {
			return true
		}
	}
	for k := range cached.Spec.Labels {
		if _, ok := candidate.Spec.Labels[k]; !ok {
			return true
		}
	}

	if candidate.NodeInfo != nil && cached.NodeInfo != nil &&
		!candidate.NodeInfo.Equal(*cached.NodeInfo) {
		return true
	}

	return false
}

func (m *Service) getNodeInfo(s SwarmService) *NodeIPSet {

	nodeInfo := NodeIPSet{}
	filter := filters.NewArgs()
	filter.Add("desired-state", "running")
	filter.Add("service", s.Spec.Name)
	taskList, err := m.DockerClient.TaskList(
		context.Background(), types.TaskListOptions{Filters: filter})
	if err != nil {
		return nil
	}

	networkName, ok := s.Spec.Labels["com.df.scrapeNetwork"]
	if !ok {
		return nil
	}

	nodeCache := map[string]string{}
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

		if nodeName, ok := nodeCache[task.NodeID]; ok {
			nodeInfo.Add(nodeName, address)
		} else {
			node, _, err := m.DockerClient.NodeInspectWithRaw(context.Background(), task.NodeID)
			if err != nil {
				continue
			}
			nodeInfo.Add(node.Description.Hostname, address)
			nodeCache[task.NodeID] = node.Description.Hostname
		}
	}

	if nodeInfo.Cardinality() == 0 {
		return nil
	}
	return &nodeInfo
}
