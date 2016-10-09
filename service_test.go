package main

import (
	"fmt"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type ServiceTestSuite struct {
	suite.Suite
	serviceName string
	removedServices []string
}

func TestServiceUnitTestSuite(t *testing.T) {
	s := new(ServiceTestSuite)
	s.serviceName = "my-service"
	s.removedServices = []string{"my-removed-service-1"}

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	suite.Run(t, s)
}

// GetServices

func (s *ServiceTestSuite) Test_GetServices_ReturnsServices() {
	services := NewService("unix:///var/run/docker.sock", "", "")

	actual, _ := services.GetServices()

	s.Equal(2, len(actual))
	index := 0
	if actual[1].Spec.Name == "util-1" {
		index = 1
	}
	s.Equal("util-1", actual[index].Spec.Name)
	s.Equal("/demo", actual[index].Spec.Labels["com.df.servicePath"])
}

func (s *ServiceTestSuite) Test_GetServices_ReturnsError_WhenNewClientFails() {
	dcOrig := dockerClient
	defer func() { dockerClient = dcOrig }()
	dockerClient = func(host string, version string, httpClient *http.Client, httpHeaders map[string]string) (*client.Client, error) {
		return &client.Client{}, fmt.Errorf("This is an error")
	}
	services := NewService("unix:///var/run/docker.sock", "", "")
	_, err := services.GetServices()
	s.Error(err)
}

func (s *ServiceTestSuite) Test_GetServices_ReturnsError_WhenServiceListFails() {
	services := NewService("unix:///this/socket/does/not/exist", "", "")

	_, err := services.GetServices()

	s.Error(err)
}

// GetNewServices

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsAllServices_WhenExecutedForTheFirstTime() {
	service := NewService("unix:///var/run/docker.sock", "", "")
	service.lastCreatedAt = time.Time{}
	services, _ := service.GetServices()

	actual, _ := service.GetNewServices(services)

	s.Equal(2, len(actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsOnlyNewServices() {
	service := NewService("unix:///var/run/docker.sock", "", "")
	services, _ := service.GetServices()

	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(0, len(actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsServices() {
	service := NewService("unix:///var/run/docker.sock", "", "")
	services, _ := service.GetServices()

	service.GetNewServices(services)

	s.Equal(2, len(service.Services))
	s.Contains(service.Services, "util-1")
	s.Contains(service.Services, "util-2")
}

// GetRemovedServices

func (s *ServiceTestSuite) Test_GetRemovedServices_ReturnsNamesOfRemovedServices() {
	service := NewService("unix:///var/run/docker.sock", "", "")
	services, _ := service.GetServices()
	service.Services["removed-service-1"] = true
	service.Services["removed-service-2"] = true

	actual := service.GetRemovedServices(services)

	s.Equal(2, len(actual))
	s.Contains(actual, "removed-service-1")
	s.Contains(actual, "removed-service-2")
}

// NotifyServicesCreate

func (s *ServiceTestSuite) Test_NotifyServicesCreate_SendsRequests() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	labels["com.df.key1"] = "value1"
	labels["label.without.correct.prefix"] = "something"

	s.verifyNotifyServiceCreate(labels, true, fmt.Sprintf("serviceName=%s&key1=value1", s.serviceName))
}

func (s *ServiceTestSuite) Test_NotifyServicesCreate_DoesNotSendRequest_WhenDfNotifyIsNotDefined() {
	labels := make(map[string]string)
	labels["DF_key1"] = "value1"

	s.verifyNotifyServiceCreate(labels, false, "")
}

func (s *ServiceTestSuite) Test_NotifyServicesCreate_ReturnsError_WhenHttpStatusIsNot200() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"

	services := NewService("unix:///var/run/docker.sock", httpSrv.URL, "")
	err := services.NotifyServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
}

func (s *ServiceTestSuite) Test_NotifyServicesCreate_ReturnsError_WhenHttpRequestReturnsError() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"

	service := NewService("unix:///var/run/docker.sock", "this-does-not-exist", "")
	err := service.NotifyServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
}

func (s *ServiceTestSuite) Test_NotifyServicesCreate_RetriesRequests() {
	attempt := 0
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 2 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		attempt = attempt + 1
	}))

	service := NewService("unix:///var/run/docker.sock", httpSrv.URL, "")
	err := service.NotifyServicesCreate(s.getSwarmServices(labels), 3, 0)

	s.NoError(err)
}

// NotifyServicesRemove

func (s *ServiceTestSuite) Test_NotifyServicesRemove_SendsRequests() {
	s.verifyNotifyServiceRemove(true, fmt.Sprintf("serviceName=%s", s.removedServices[0]))
}

func (s *ServiceTestSuite) Test_NotifyServicesRemove_ReturnsError_WhenHttpStatusIsNot200() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	services := NewService("unix:///var/run/docker.sock", "", httpSrv.URL)
	err := services.NotifyServicesRemove(s.removedServices, 1, 0)

	s.Error(err)
}

func (s *ServiceTestSuite) Test_NotifyServicesRemove_ReturnsError_WhenHttpRequestReturnsError() {
	service := NewService("unix:///var/run/docker.sock", "", "this-does-not-exist")
	err := service.NotifyServicesRemove(s.removedServices, 1, 0)

	s.Error(err)
}

func (s *ServiceTestSuite) Test_NotifyServicesRemove_RetriesRequests() {
	attempt := 0
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 2 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		attempt = attempt + 1
	}))

	service := NewService("unix:///var/run/docker.sock", "", httpSrv.URL)
	err := service.NotifyServicesRemove(s.removedServices, 3, 0)

	s.NoError(err)
}

// NewService

func (s *ServiceTestSuite) Test_NewService_SetsHost() {
	expected := "this-is-a-host"

	service := NewService(expected, "", "")

	s.Equal(expected, service.Host)
}

func (s *ServiceTestSuite) Test_NewService_SetsNotifUrl() {
	expected := "this-is-a-notification-url"

	service := NewService("", expected, "")

	s.Equal(expected, service.NotifCreateServiceUrl)
}

// NewServiceFromEnv

func (s *ServiceTestSuite) Test_NewServiceFromEnv_SetsHost() {
	host := os.Getenv("DF_DOCKER_HOST")
	defer func() { os.Setenv("DF_DOCKER_HOST", host) }()
	expected := "this-is-a-host"
	os.Setenv("DF_DOCKER_HOST", expected)

	service := NewServiceFromEnv()

	s.Equal(expected, service.Host)
}

func (s *ServiceTestSuite) Test_NewServiceFromEnv_SetsHostToSocket_WhenEnvIsNotPresent() {
	host := os.Getenv("DF_DOCKER_HOST")
	defer func() { os.Setenv("DF_DOCKER_HOST", host) }()
	os.Unsetenv("DF_DOCKER_HOST")

	service := NewServiceFromEnv()

	s.Equal("unix:///var/run/docker.sock", service.Host)
}

func (s *ServiceTestSuite) Test_NewServiceFromEnv_SetsNotifUrl() {
	host := os.Getenv("DF_NOTIFICATION_URL")
	defer func() { os.Setenv("DF_NOTIFICATION_URL", host) }()
	expected := "this-is-a-notification-url"
	os.Setenv("DF_NOTIFICATION_URL", expected)

	service := NewServiceFromEnv()

	s.Equal(expected, service.NotifCreateServiceUrl)
}

func (s *ServiceTestSuite) Test_NewServiceFromEnv_SetsNotifCreateServiceUrl() {
	host := os.Getenv("DF_NOTIF_CREATE_SERVICE_URL")
	defer func() { os.Setenv("DF_NOTIF_CREATE_SERVICE_URL", host) }()
	expected := "this-is-a-notification-url"
	os.Setenv("DF_NOTIF_CREATE_SERVICE_URL", expected)

	service := NewServiceFromEnv()

	s.Equal(expected, service.NotifCreateServiceUrl)
}

func (s *ServiceTestSuite) Test_NewServiceFromEnv_SetsNotifRemoveServiceUrl() {
	host := os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL")
	defer func() { os.Setenv("DF_NOTIF_REMOVE_SERVICE_URL", host) }()
	expected := "this-is-a-notification-url"
	os.Setenv("DF_NOTIF_REMOVE_SERVICE_URL", expected)

	service := NewServiceFromEnv()

	s.Equal(expected, service.NotifRemoveServiceUrl)
}

// Util

func (s *ServiceTestSuite) verifyNotifyServiceCreate(labels map[string]string, expectSent bool, expectQuery string) {
	actualSent := false
	actualQuery := ""
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/docker-flow-proxy/reconfigure":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				actualQuery = r.URL.RawQuery
				actualSent = true
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer func() { httpSrv.Close() }()
	url := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	services := NewService("unix:///var/run/docker.sock", url, "")
	err := services.NotifyServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.NoError(err)
	s.Equal(expectSent, actualSent)
	if expectSent {
		s.Equal(expectQuery, actualQuery)
	}
}

func (s *ServiceTestSuite) verifyNotifyServiceRemove(expectSent bool, expectQuery string) {
	actualSent := false
	actualQuery := ""
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/docker-flow-proxy/remove":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				actualQuery = r.URL.RawQuery
				actualSent = true
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer func() { httpSrv.Close() }()
	url := fmt.Sprintf("%s/v1/docker-flow-proxy/remove", httpSrv.URL)

	service := NewService("unix:///var/run/docker.sock", "", url)
	service.Services[s.removedServices[0]] = true
	err := service.NotifyServicesRemove(s.removedServices, 1, 0)

	s.NoError(err)
	s.Equal(expectSent, actualSent)
	if expectSent {
		s.Equal(expectQuery, actualQuery)
		s.NotContains(service.Services, s.removedServices[0])
	}
}

func (s *ServiceTestSuite) getSwarmServices(labels map[string]string) []swarm.Service {
	ann := swarm.Annotations{
		Name:   s.serviceName,
		Labels: labels,
	}
	spec := swarm.ServiceSpec{
		Annotations: ann,
	}
	serv := swarm.Service{
		Spec: spec,
	}
	return []swarm.Service{serv}
}
