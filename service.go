package main

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var lastCreatedAt time.Time
var logPrintf = log.Printf
var dockerClient = client.NewClient

type Service struct {
	Host     string
	NotifUrl string
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

	return services, nil
}

func (m *Service) GetNewServices() ([]swarm.Service, error) {
	services, err := m.GetServices()
	if err != nil {
		logPrintf(err.Error())
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

func (m *Service) NotifyServices(services []swarm.Service, retries, interval int) error {
	errs := []error{}
	for _, s := range services {
		fullUrl := fmt.Sprintf("%s?serviceName=%s", m.NotifUrl, s.Spec.Name)
		if _, ok := s.Spec.Labels["com.df.notify"]; ok {
			for k, v := range s.Spec.Labels {
				if strings.HasPrefix(k, "com.df") && k != "com.df.notify" {
					fullUrl = fmt.Sprintf("%s&%s=%s", fullUrl, strings.TrimLeft(k, "com.df."), v)
				}
			}
			logPrintf("Sending a service notification to %s", fullUrl)
			for i := 1; i <= retries; i++ {
				resp, err := http.Get(fullUrl)
				if err == nil && resp.StatusCode == http.StatusOK {
					break
				} else if i < retries {
					logPrintf("Notification to %s failed. Retrying...", fullUrl)
					if interval > 0 {
						t := time.NewTicker(time.Second * time.Duration(interval))
						<-t.C
					}
				} else {
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
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("At least one request produced errors. Please consult logs for more details.")
	}
	return nil
}

func NewService(host, notifUrl string) Service {
	return Service{
		Host:     host,
		NotifUrl: notifUrl,
	}
}

func NewServiceFromEnv() Service {
	host := "unix:///var/run/docker.sock"
	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	return Service{
		Host:     host,
		NotifUrl: os.Getenv("DF_NOTIFICATION_URL"),
	}
}
