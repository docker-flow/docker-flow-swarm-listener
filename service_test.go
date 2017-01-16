package main

import (
	"fmt"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type ServiceTestSuite struct {
	suite.Suite
	serviceName     string
	removedServices []string
}

func TestServiceUnitTestSuite(t *testing.T) {
	s := new(ServiceTestSuite)
	s.serviceName = "my-service"
	s.removedServices = []string{"my-removed-service-1"}

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	createTestServices()
	suite.Run(t, s)
	removeTestServices()
}

// GetServices

func (s *ServiceTestSuite) Test_GetServices_ReturnsServices() {
	services := NewService("unix:///var/run/docker.sock", "", "")

	actual, _ := services.GetServices()

	s.Equal(1, len(actual))
	s.Equal("util-1", actual[0].Spec.Name)
	s.Equal("/demo", actual[0].Spec.Labels["com.df.servicePath"])
	s.Equal("true", actual[0].Spec.Labels["com.df.distribute"])
}

//func (s *ServiceTestSuite) Test_GetServices_ReturnsError_WhenNewClientFails() {
//	services := NewService("unix:///var/run/docker.sock", "", "")
//	hostOrig := services.Host
//	defer func() { services.Host = hostOrig }()
//	services.Host = "This host does not exist"
//	_, err := services.GetServices()
//	s.Error(err)
//}

func (s *ServiceTestSuite) Test_GetServices_ReturnsError_WhenServiceListFails() {
	services := NewService("unix:///this/socket/does/not/exist", "", "")

	_, err := services.GetServices()

	s.Error(err)
}

// GetNewServices

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsAllServices_WhenExecutedForTheFirstTime() {
	service := NewService("unix:///var/run/docker.sock", "", "")
	service.ServiceLastCreatedAt = time.Time{}
	services, _ := service.GetServices()

	actual, _ := service.GetNewServices(services)

	s.Equal(1, len(actual))
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

	s.Equal(1, len(service.Services))
	s.Contains(service.Services, "util-1")
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
	labels["com.df.distribute"] = "true"
	labels["label.without.correct.prefix"] = "something"

	s.verifyNotifyServiceCreate(labels, true, fmt.Sprintf("distribute=true&serviceName=%s", s.serviceName))
}

func (s *ServiceTestSuite) Test_NotifyServicesCreate_ReturnsError_WhenUrlCannotBeParsed() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	services := NewService("unix:///var/run/docker.sock", "%%%", "")
	err := services.NotifyServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
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
	s.verifyNotifyServiceRemove(true, fmt.Sprintf("distribute=true&serviceName=%s", s.removedServices[0]))
}

func (s *ServiceTestSuite) Test_NotifyServicesRemove_ReturnsError_WhenUrlCannotBeParsed() {
	services := NewService("unix:///var/run/docker.sock", "", "%%%")
	err := services.NotifyServicesRemove(s.removedServices, 1, 0)

	s.Error(err)
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

	s.Equal(expected, service.NotifyCreateServiceUrl)
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

	s.Equal(expected, service.NotifyCreateServiceUrl)
}

func (s *ServiceTestSuite) Test_NewServiceFromEnv_SetsNotifyCreateServiceUrl() {
	host := os.Getenv("DF_NOTIFY_CREATE_SERVICE_URL")
	defer func() { os.Setenv("DF_NOTIFY_CREATE_SERVICE_URL", host) }()
	expected := "this-is-a-notification-url"
	os.Setenv("DF_NOTIFY_CREATE_SERVICE_URL", expected)

	service := NewServiceFromEnv()

	s.Equal(expected, service.NotifyCreateServiceUrl)
}

func (s *ServiceTestSuite) Test_NewServiceFromEnv_SetsNotifyRemoveServiceUrl() {
	host := os.Getenv("DF_NOTIFY_REMOVE_SERVICE_URL")
	defer func() { os.Setenv("DF_NOTIFY_REMOVE_SERVICE_URL", host) }()
	expected := "this-is-a-notification-url"
	os.Setenv("DF_NOTIFY_REMOVE_SERVICE_URL", expected)

	service := NewServiceFromEnv()

	s.Equal(expected, service.NotifyRemoveServiceUrl)
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

func createTestServices() {
	createTestService("util-1", []string{"com.df.notify=true", "com.df.servicePath=/demo", "com.df.distribute=true"})
	createTestService("util-2", []string{})
}

func createTestService(name string, labels []string) {
	args := []string{"service", "create", "--name", name}
	for _, v := range labels {
		args = append(args, "-l", v)
	}
	args = append(args, "alpine", "sleep", "1000000000")
	exec.Command("docker", args...).Output()
}

func removeTestServices() {
	removeTestService("util-1")
	removeTestService("util-2")
}

func removeTestService(name string) {
	exec.Command("docker", "service", "rm", name).Output()
}

// Mocks

type ServicerMock struct {
	mock.Mock
}

func (m *ServicerMock) Execute(args []string) error {
	params := m.Called(args)
	return params.Error(0)
}

func (m *ServicerMock) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.Called(w, req)
}

func (m *ServicerMock) GetServices() ([]swarm.Service, error) {
	args := m.Called()
	return args.Get(0).([]swarm.Service), args.Error(1)
}

func (m *ServicerMock) GetNewServices(services []swarm.Service) ([]swarm.Service, error) {
	args := m.Called()
	return args.Get(0).([]swarm.Service), args.Error(1)
}

func (m *ServicerMock) NotifyServicesCreate(services []swarm.Service, retries, interval int) error {
	args := m.Called(services, retries, interval)
	return args.Error(0)
}

func (m *ServicerMock) NotifyServicesRemove(services []string, retries, interval int) error {
	args := m.Called(services, retries, interval)
	return args.Error(0)
}

func getServicerMock(skipMethod string) *ServicerMock {
	mockObj := new(ServicerMock)
	if !strings.EqualFold("GetServices", skipMethod) {
		mockObj.On("GetServices").Return([]swarm.Service{}, nil)
	}
	if !strings.EqualFold("GetNewServices", skipMethod) {
		mockObj.On("GetNewServices", mock.Anything).Return([]swarm.Service{}, nil)
	}
	if !strings.EqualFold("NotifyServicesCreate", skipMethod) {
		mockObj.On("NotifyServicesCreate", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}
	if !strings.EqualFold("NotifyServicesRemove", skipMethod) {
		mockObj.On("NotifyServicesRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}
	return mockObj
}
