package main

import (
	"./service"
	"fmt"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"strings"
	"testing"
	"time"
)

type ServerTestSuite struct {
	suite.Suite
}

func (s *ServerTestSuite) SetupTest() {
}

func TestServerUnitTestSuite(t *testing.T) {
	s := new(ServerTestSuite)
	suite.Run(t, s)
}

// Run

func (s *ServerTestSuite) Test_Run_InvokesHTTPListenAndServe() {
	var actual string
	expected := fmt.Sprintf(":8080")
	httpListenAndServe = func(addr string, handler http.Handler) error {
		actual = addr
		return nil
	}

	serve := Serve{}
	serve.Run()

	s.Equal(expected, actual)
}

func (s *ServerTestSuite) Test_Run_ReturnsError_WhenHTTPListenAndServeFails() {
	orig := httpListenAndServe
	defer func() {
		httpListenAndServe = orig
	}()
	httpListenAndServe = func(addr string, handler http.Handler) error {
		return fmt.Errorf("This is an error")
	}

	serve := Serve{}
	actual := serve.Run()

	s.Error(actual)
}

// ServeHTTP

func (s *ServerTestSuite) Test_ServeHTTP_SetsContentTypeToJSON_WhenUrlIsNotifyServices() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	notifMock := NotificationMock{
		ServicesCreateMock: func(services *[]swarm.Service, retries, interval int) error {
			return nil
		},
	}

	srv := NewServe(getServicerMock(""), notifMock)
	srv.ServeHTTP(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatusOK_WhenUrlIsNotifyServices() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	rw := getResponseWriterMock()
	notifMock := NotificationMock{
		ServicesCreateMock: func(services *[]swarm.Service, retries, interval int) error {
			return nil
		},
	}

	srv := NewServe(getServicerMock(""), notifMock)
	srv.ServeHTTP(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatusNotFound_WhenUrlIsUnknown() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/this-api-does-not-exist", nil)
	rw := getResponseWriterMock()
	notifMock := NotificationMock{}

	srv := NewServe(getServicerMock(""), notifMock)
	srv.ServeHTTP(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 404)
}

func (s *ServerTestSuite) Test_ServeHTTP_InvokesNotifyServicesCreate_WhenUrlIsNotifyservices() {
	servicerMock := getServicerMock("GetServices")
	service1 := swarm.Service{
		ID: "my-service-id-1",
	}
	expectedServices := []swarm.Service{service1}
	servicerMock.On("GetServices").Return(expectedServices, nil)
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	rw := getResponseWriterMock()
	actualServices := []swarm.Service{}
	actualRetries := 0
	actualInterval := 0
	notifMock := NotificationMock{
		ServicesCreateMock: func(services *[]swarm.Service, retries, interval int) error {
			actualServices = *services
			actualRetries = retries
			actualInterval = interval
			return nil
		},
	}

	srv := NewServe(servicerMock, notifMock)
	srv.ServeHTTP(rw, req)

	time.Sleep(1 * time.Millisecond)
	s.Equal(expectedServices, actualServices)
	s.Equal(10, actualRetries)
	s.Equal(5, actualInterval)
}

// NewServe

func (s *ServerTestSuite) Test_NewServe_SetsService() {
	srv := service.NewServiceFromEnv()
	notifMock := NotificationMock{}
	serve := NewServe(srv, notifMock)

	s.Equal(srv, serve.Service)
}

func (s *ServerTestSuite) Test_NewServe_SetsNotifier() {
	srv := service.NewServiceFromEnv()
	notifMock := NotificationMock{}
	serve := NewServe(srv, notifMock)

	s.Equal(notifMock, serve.Notification)
}

// Mocks

type ResponseWriterMock struct {
	mock.Mock
}

func (m *ResponseWriterMock) Header() http.Header {
	m.Called()
	return make(map[string][]string)
}

func (m *ResponseWriterMock) Write(data []byte) (int, error) {
	params := m.Called(data)
	return params.Int(0), params.Error(1)
}

func (m *ResponseWriterMock) WriteHeader(header int) {
	m.Called(header)
}

func getResponseWriterMock() *ResponseWriterMock {
	mockObj := new(ResponseWriterMock)
	mockObj.On("Header").Return(nil)
	mockObj.On("Write", mock.Anything).Return(0, nil)
	mockObj.On("WriteHeader", mock.Anything)
	return mockObj
}

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

func (m *ServicerMock) GetServices() (*[]swarm.Service, error) {
	args := m.Called()
	s := args.Get(0).([]swarm.Service)
	return &s, args.Error(1)
}

func (m *ServicerMock) GetNewServices(services *[]swarm.Service) (*[]swarm.Service, error) {
	args := m.Called()
	return args.Get(0).(*[]swarm.Service), args.Error(1)
}

func (m *ServicerMock) GetRemovedServices(services *[]swarm.Service) *[]string {
	args := m.Called(services)
	return args.Get(0).(*[]string)
}

func getServicerMock(skipMethod string) *ServicerMock {
	mockObj := new(ServicerMock)
	if !strings.EqualFold("GetServices", skipMethod) {
		mockObj.On("GetServices").Return([]swarm.Service{}, nil)
	}
	if !strings.EqualFold("GetNewServices", skipMethod) {
		mockObj.On("GetNewServices", mock.Anything).Return([]swarm.Service{}, nil)
	}
	if !strings.EqualFold("GetRemovedServices", skipMethod) {
		mockObj.On("GetRemovedServices", mock.Anything).Return(&[]string{})
	}
	return mockObj
}

type NotificationMock struct {
	ServicesCreateMock func(services *[]swarm.Service, retries, interval int) error
	ServicesRemoveMock func(remove *[]string, retries, interval int) error
}

func (m NotificationMock) ServicesCreate(services *[]swarm.Service, retries, interval int) error {
	return m.ServicesCreateMock(services, retries, interval)
}

func (m NotificationMock) ServicesRemove(remove *[]string, retries, interval int) error {
	return m.ServicesRemoveMock(remove, retries, interval)
}
