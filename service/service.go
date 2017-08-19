package service

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"os"
	"strings"
	"time"
)

var Services map[string]SwarmService

type Service struct {
	Host                 string
	ServiceLastUpdatedAt time.Time
	DockerClient         *client.Client
}

type Servicer interface {
	GetServices() (*[]SwarmService, error)
	GetNewServices(services *[]SwarmService) (*[]SwarmService, error)
	GetRemovedServices(services *[]SwarmService) *[]string
	GetServicesParameters(services *[]SwarmService) *[]map[string]string
}

func (m *Service) GetServicesParameters(services *[]SwarmService) *[]map[string]string {
	var parameters = []map[string]string{}
	for _, s := range *services {
		if _, ok := s.Spec.Labels[os.Getenv("DF_NOTIFY_LABEL")]; ok {
			var sParams = make(map[string]string)
			sParams["serviceName"] = s.Spec.Name
			for k, v := range s.Spec.Labels {
				if strings.HasPrefix(k, "com.df") && k != os.Getenv("DF_NOTIFY_LABEL") {
					sParams[strings.TrimPrefix(k, "com.df.")] = v
				}
			}
			parameters = append(parameters, sParams)
		}
	}
	return &parameters
}

func (m *Service) GetServices() (*[]SwarmService, error) {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("%s=true", os.Getenv("DF_NOTIFY_LABEL")))
	services, err := m.DockerClient.ServiceList(context.Background(), types.ServiceListOptions{Filters: filter})
	if err != nil {
		logPrintf(err.Error())
		return &[]SwarmService{}, err
	}
	swarmServices := []SwarmService{}
	for _, s := range services {
		swarmServices = append(swarmServices, SwarmService{s})
	}
	return &swarmServices, nil
}

func (m *Service) GetNewServices(services *[]SwarmService) (*[]SwarmService, error) {
	newServices := []SwarmService{}
	tmpUpdatedAt := m.ServiceLastUpdatedAt
	for _, s := range *services {
		if tmpUpdatedAt.Nanosecond() == 0 || s.Meta.UpdatedAt.After(tmpUpdatedAt) {
			updated := false
			if service, ok := Services[s.Spec.Name]; ok {
				// Check whether a label was added or updated
				for k, v := range s.Spec.Labels {
					if strings.HasPrefix(k, "com.df.") {
						if storedValue, ok := service.Spec.Labels[k]; !ok || v != storedValue {
							updated = true
						}
					}
				}
				// Check whether a label was removed
				for k := range service.Spec.Labels {
					if _, ok := s.Spec.Labels[k]; !ok {
						updated = true
					}
				}
			} else { // It's a new service
				updated = true
			}
			if updated {
				newServices = append(newServices, s)
				Services[s.Spec.Name] = s
				if m.ServiceLastUpdatedAt.Before(s.Meta.UpdatedAt) {
					m.ServiceLastUpdatedAt = s.Meta.UpdatedAt
				}
			}
		}
	}
	return &newServices, nil
}

func (m *Service) GetRemovedServices(services *[]SwarmService) *[]string {
	tmpMap := make(map[string]SwarmService)
	for k, v := range Services {
		tmpMap[k] = v
	}
	for _, v := range *services {
		if _, ok := Services[v.Spec.Name]; ok {
			delete(tmpMap, v.Spec.Name)
		}
	}
	rs := []string{}
	for k := range tmpMap {
		rs = append(rs, k)
	}
	return &rs
}

func NewService(host string) *Service {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	dc, err := client.NewClient(host, dockerApiVersion, nil, defaultHeaders)
	if err != nil {
		logPrintf(err.Error())
	}
	Services = make(map[string]SwarmService)
	return &Service{
		Host:         host,
		DockerClient: dc,
	}
}

func NewServiceFromEnv() *Service {
	host := "unix:///var/run/docker.sock"
	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	return NewService(host)
}
