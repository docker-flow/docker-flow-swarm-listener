package main

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
//	"time"
	"net/http"
	"fmt"
	"strings"
	"log"
)

type Service struct{}

var dockerClient = client.NewClient

func (m *Service) GetServices(host string) ([]swarm.Service, error) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	// TODO: Move to main
	dc, err := dockerClient(host, "v1.22", nil, defaultHeaders)

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
			resp, err := http.Get(fullUrl)
			if err != nil {
				log.Printf("ERROR: %s", err.Error())
				errs = append(errs, err)
			} else if resp.StatusCode != http.StatusOK {
				msg := fmt.Errorf("Request %s returned status code %d", fullUrl, resp.StatusCode)
				log.Printf("ERROR: %s", msg)
				errs = append(errs, msg)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("Requests produced errors")
	}
	return nil
}

func NewServices() Service {
	return Service{}
}
