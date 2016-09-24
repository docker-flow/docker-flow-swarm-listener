package main

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"time"
	"net/http"
	"fmt"
	"strings"
	"log"
)

var lastCreatedAt time.Time
var logPrintf = log.Printf
var dockerClient = client.NewClient
type Service struct{
	Host string
}

func (m *Service) GetServices() ([]swarm.Service, error) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	// TODO: Move to main
	dc, err := dockerClient(m.Host, "v1.22", nil, defaultHeaders)

	if err != nil {
		return []swarm.Service{}, err
	}

	services, err := dc.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		return []swarm.Service{}, err
	}

//	t := time.NewTicker(time.Second * 5)
//	for {
//		println("Something")
//		<-t.C
//	}

	return services, nil
}

func (m *Service) GetNewServices() ([]swarm.Service, error) {
	services, err := m.GetServices()
	if err != nil {
		return []swarm.Service{}, err
	}
	newServices := []swarm.Service{}
	tmpCreatedAt := lastCreatedAt
	for _, s := range services {
		if tmpCreatedAt.Nanosecond() == 0 || s.Meta.CreatedAt.After(tmpCreatedAt) {
			newServices = append(newServices, s)
			if lastCreatedAt.Before(s.Meta.CreatedAt) {
				lastCreatedAt = s.Meta.CreatedAt
			}
		}
	}
	return newServices, nil
}

func (m *Service) NotifyServices(services []swarm.Service, url string) error {
	errs := []error{}
	for _, s := range services {
		fullUrl := fmt.Sprintf("%s?serviceName=%s", url, s.Spec.Name)
		if _, ok := s.Spec.Labels["DF_NOTIFY"]; ok {
			for k, v := range s.Spec.Labels {
				if strings.HasPrefix(k, "DF_") && k != "DF_NOTIFY" {
					fullUrl = fmt.Sprintf("%s&%s=%s", fullUrl, strings.TrimLeft(k, "DF_"), v)
				}
			}
			logPrintf("Sending a service notification to %s", fullUrl)
			resp, err := http.Get(fullUrl)
			if err != nil {
				logPrintf("ERROR: %s", err.Error())
				errs = append(errs, err)
			} else if resp.StatusCode != http.StatusOK {
				msg := fmt.Errorf("Request %s returned status code %d", fullUrl, resp.StatusCode)
				logPrintf("ERROR: %s", msg)
				errs = append(errs, msg)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("At least one request produced errors. Please consult logs for more details.")
	}
	return nil
}

func NewServices(host string) Service {
	return Service{
		Host: host,
	}
}
