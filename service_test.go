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
)

type ServiceTestSuite struct {
	suite.Suite
	serviceName string
}

func TestServiceUnitTestSuite(t *testing.T) {
	s := new(ServiceTestSuite)
	s.serviceName = "my-service"

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	suite.Run(t, s)
}

// GetServices

func (s *ServiceTestSuite) Test_GetServices_ReturnsServices() {
	services := NewService("unix:///var/run/docker.sock", "")

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
	services := NewService("unix:///var/run/docker.sock", "")
	_, err := services.GetServices()
	s.Error(err)
}

func (s *ServiceTestSuite) Test_GetServices_ReturnsError_WhenServiceListFails() {
	services := NewService("unix:///this/socket/does/not/exist", "")

	_, err := services.GetServices()

	s.Error(err)
}

// GetNewServices

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsAllServices_WhenExecutedForTheFirstTime() {
	services := NewService("unix:///var/run/docker.sock", "")

	actual, _ := services.GetNewServices()

	s.Equal(2, len(actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsError_WhenGetServicesFails() {
	services := NewService("unix:///this/socket/does/not/exist", "")

	_, err := services.GetNewServices()

	s.Error(err)
}

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsOnlyNewServices() {
	services := NewService("unix:///var/run/docker.sock", "")

	services.GetNewServices()
	actual, _ := services.GetNewServices()

	s.Equal(0, len(actual))
}

// NotifyServices

func (s *ServiceTestSuite) Test_NotifyServices_SendsRequests() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	labels["com.df.key1"] = "value1"
	labels["label.without.correct.prefix"] = "something"

	s.verifyNotifyService(labels, true, fmt.Sprintf("serviceName=%s&key1=value1", s.serviceName))
}

func (s *ServiceTestSuite) Test_NotifyServices_DoesNotSendRequest_WhenDfNotifyIsNotDefined() {
	labels := make(map[string]string)
	labels["DF_key1"] = "value1"

	s.verifyNotifyService(labels, false, "")
}

func (s *ServiceTestSuite) Test_NotifyServices_ReturnsError_WhenHttpStatusIsNot200() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"

	services := NewService("unix:///var/run/docker.sock", httpSrv.URL)
	err := services.NotifyServices(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
}

func (s *ServiceTestSuite) Test_NotifyServices_ReturnsError_WhenHttpRequestReturnsError() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"

	service := NewService("unix:///var/run/docker.sock", "this-does-not-exist")
	err := service.NotifyServices(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
}

func (s *ServiceTestSuite) Test_NotifyServices_RetriesRequests() {
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

	service := NewService("unix:///var/run/docker.sock", httpSrv.URL)
	err := service.NotifyServices(s.getSwarmServices(labels), 3, 0)

	s.NoError(err)
}

// NewService

func (s *ServiceTestSuite) Test_NewService_SetsHost() {
	expected := "this-is-a-host"

	service := NewService(expected, "")

	s.Equal(expected, service.Host)
}

func (s *ServiceTestSuite) Test_NewService_SetsNotifUrl() {
	expected := "this-is-a-notification-url"

	service := NewService("", expected)

	s.Equal(expected, service.NotifUrl)
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

	s.Equal(expected, service.NotifUrl)
}

// Util

func (s *ServiceTestSuite) verifyNotifyService(labels map[string]string, expectSent bool, expectQuery string) {
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

	services := NewService("unix:///var/run/docker.sock", url)
	err := services.NotifyServices(s.getSwarmServices(labels), 1, 0)

	s.NoError(err)
	s.Equal(expectSent, actualSent)
	if expectSent {
		s.Equal(expectQuery, actualQuery)
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
