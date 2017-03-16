package main

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var logPrintf = log.Printf
var dockerApiVersion string = "v1.22"

type Service struct {
	Host                   string
	NotifyCreateServiceUrl []string
	NotifyRemoveServiceUrl []string
	Services               map[string]swarm.Service
	ServiceLastUpdatedAt   time.Time
	DockerClient           *client.Client
}

type Servicer interface {
	GetServices() ([]swarm.Service, error)
	GetNewServices(services []swarm.Service) ([]swarm.Service, error)
	NotifyServicesCreate(services []swarm.Service, retries, interval int) error
	NotifyServicesRemove(services []string, retries, interval int) error
}

func (m *Service) GetServices() ([]swarm.Service, error) {
	filter := filters.NewArgs()
	filter.Add("label", "com.df.notify=true")
	services, err := m.DockerClient.ServiceList(context.Background(), types.ServiceListOptions{Filters: filter})
	if err != nil {
		logPrintf(err.Error())
		return []swarm.Service{}, err
	}
	return services, nil
}

func (m *Service) GetNewServices(services []swarm.Service) ([]swarm.Service, error) {
	newServices := []swarm.Service{}
	tmpUpdatedAt := m.ServiceLastUpdatedAt
	for _, s := range services {
		if tmpUpdatedAt.Nanosecond() == 0 || s.Meta.UpdatedAt.After(tmpUpdatedAt) {
			updated := false
			if service, ok := m.Services[s.Spec.Name]; ok {
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
				m.Services[s.Spec.Name] = s
				if m.ServiceLastUpdatedAt.Before(s.Meta.UpdatedAt) {
					m.ServiceLastUpdatedAt = s.Meta.UpdatedAt
				}
			}
		}
	}
	return newServices, nil
}

func (m *Service) GetRemovedServices(services []swarm.Service) []string {
	tmpMap := make(map[string]swarm.Service)
	for k, v := range m.Services {
		tmpMap[k] = v
	}
	for _, v := range services {
		if _, ok := m.Services[v.Spec.Name]; ok {
			delete(tmpMap, v.Spec.Name)
		}
	}
	rs := []string{}
	for k := range tmpMap {
		rs = append(rs, k)
	}
	return rs
}

func (m *Service) NotifyServicesCreate(services []swarm.Service, retries, interval int) error {
	errs := []error{}
	for _, s := range services {
		if _, ok := s.Spec.Labels["com.df.notify"]; ok {
			parameters := url.Values{}
			parameters.Add("serviceName", s.Spec.Name)
			for k, v := range s.Spec.Labels {
				if strings.HasPrefix(k, "com.df") && k != "com.df.notify" {
					parameters.Add(strings.TrimPrefix(k, "com.df."), v)
				}
			}
			for _, addr := range m.NotifyCreateServiceUrl {
				urlObj, err := url.Parse(addr)
				if err != nil {
					logPrintf("ERROR: %s", err.Error())
					errs = append(errs, err)
					break
				}
				urlObj.RawQuery = parameters.Encode()
				fullUrl := urlObj.String()
				logPrintf("Sending service created notification to %s", fullUrl)
				for i := 1; i <= retries; i++ {
					resp, err := http.Get(fullUrl)
					if err == nil && resp.StatusCode == http.StatusOK {
						break
					} else if i < retries {
						if interval > 0 {
							t := time.NewTicker(time.Second * time.Duration(interval))
							<-t.C
						}
					} else {
						if err != nil {
							logPrintf("ERROR: %s", err.Error())
							errs = append(errs, err)
						} else if resp.StatusCode != http.StatusOK {
							body, _ := ioutil.ReadAll(resp.Body)
							msg := fmt.Errorf("Request %s returned status code %d\n%s", fullUrl, resp.StatusCode, string(body[:]))
							logPrintf("ERROR: %s", msg)
							errs = append(errs, msg)
						}
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

func (m *Service) NotifyServicesRemove(services []string, retries, interval int) error {
	errs := []error{}
	for _, v := range services {
		parameters := url.Values{}
		parameters.Add("serviceName", v)
		parameters.Add("distribute", "true")
		for _, addr := range m.NotifyRemoveServiceUrl {
			urlObj, err := url.Parse(addr)
			if err != nil {
				logPrintf("ERROR: %s", err.Error())
				errs = append(errs, err)
				break
			}
			urlObj.RawQuery = parameters.Encode()
			fullUrl := urlObj.String()
			logPrintf("Sending service removed notification to %s", fullUrl)
			for i := 1; i <= retries; i++ {
				resp, err := http.Get(fullUrl)
				if err == nil && resp.StatusCode == http.StatusOK {
					delete(m.Services, v)
					break
				} else if i < retries {
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

func NewService(host string, notifyCreateServiceUrl, notifyRemoveServiceUrl []string) *Service {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	dc, err := client.NewClient(host, dockerApiVersion, nil, defaultHeaders)
	if err != nil {
		logPrintf(err.Error())
	}
	return &Service{
		Host: host,
		NotifyCreateServiceUrl: notifyCreateServiceUrl,
		NotifyRemoveServiceUrl: notifyRemoveServiceUrl,
		Services:               make(map[string]swarm.Service),
		DockerClient:           dc,
	}
}

func NewServiceFromEnv() *Service {
	var notifyCreateServiceUrl []string
	var notifyRemoveServiceUrl []string
	host := "unix:///var/run/docker.sock"

	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	if len(os.Getenv("DF_NOTIFY_CREATE_SERVICE_URL")) > 0 {
		notifyCreateServiceUrl = strings.Split(os.Getenv("DF_NOTIFY_CREATE_SERVICE_URL"), ",")
	} else if len(os.Getenv("DF_NOTIF_CREATE_SERVICE_URL")) > 0 { // Deprecated since dec. 2016
		notifyCreateServiceUrl = strings.Split(os.Getenv("DF_NOTIF_CREATE_SERVICE_URL"), ",")
	} else {
		notifyCreateServiceUrl = strings.Split(os.Getenv("DF_NOTIFICATION_URL"), ",")
	}
	if len(os.Getenv("DF_NOTIFY_REMOVE_SERVICE_URL")) > 0 {
		notifyRemoveServiceUrl = strings.Split(os.Getenv("DF_NOTIFY_REMOVE_SERVICE_URL"), ",")
	} else if len(os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL")) > 0 { // Deprecated since dec. 2016
		notifyRemoveServiceUrl = strings.Split(os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL"), ",")
	} else {
		notifyRemoveServiceUrl = strings.Split(os.Getenv("DF_NOTIFICATION_URL"), ",")
	}
	return NewService(host, notifyCreateServiceUrl, notifyRemoveServiceUrl)
}
