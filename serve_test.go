package main

import (
	"./service"
	"encoding/json"
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

// NotifyServices

func (s *ServerTestSuite) Test_NotifyServices_ReturnsStatus200() {
	servicerMock := getServicerMock("")
	notifMock := NotificationMock{
		ServicesCreateMock: func(services *[]service.SwarmService, retries, interval int) error {
			return nil
		},
	}
	rw := getResponseWriterMock()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	expected, _ := json.Marshal(Response{Status: "OK"})

	srv := NewServe(servicerMock, notifMock)
	srv.NotifyServices(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
	rw.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_NotifyServices_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	notifMock := NotificationMock{
		ServicesCreateMock: func(services *[]service.SwarmService, retries, interval int) error {
			return nil
		},
	}

	srv := NewServe(getServicerMock(""), notifMock)
	srv.NotifyServices(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_NotifyServices_InvokesServicesCreate() {
	servicerMock := getServicerMock("GetServices")
	service1 := swarm.Service{
		ID: "my-service-id-1",
	}
	expectedServices := []service.SwarmService{{service1}}
	servicerMock.On("GetServices").Return(expectedServices, nil)
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	rw := getResponseWriterMock()
	actualServices := []service.SwarmService{}
	actualRetries := 0
	actualInterval := 0
	notifMock := NotificationMock{
		ServicesCreateMock: func(services *[]service.SwarmService, retries, interval int) error {
			actualServices = *services
			actualRetries = retries
			actualInterval = interval
			return nil
		},
	}

	srv := NewServe(servicerMock, notifMock)
	srv.NotifyServices(rw, req)

	time.Sleep(1 * time.Millisecond)
	s.Equal(expectedServices, actualServices)
	s.Equal(10, actualRetries)
	s.Equal(5, actualInterval)
}

// GetServices

func (s *ServerTestSuite) Test_GetServices_ReturnsServices() {
	servicerMock := getServicerMock("GetServicesParameters")
	mapParam := []map[string]string{
		{"serviceName": "demo",
			"notify":      "true",
			"servicePath": "/demo",
			"distribute":  "true"},
	}
	servicerMock.On("GetServicesParameters", mock.Anything).Return(&mapParam)
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/get-services", nil)
	rw := getResponseWriterMock()
	notifMock := NotificationMock{}
	srv := NewServe(servicerMock, notifMock)
	srv.GetServices(rw, req)

	call := rw.GetLastMethodCall("Write")
	value, _ := call.Arguments.Get(0).([]byte)
	rsp := []map[string]string{}
	json.Unmarshal(value, &rsp)
	s.Equal(&mapParam, &rsp)
}

// PingHandler

func (s *ServerTestSuite) Test_PingHandler_ReturnsStatus200() {
	servicerMock := getServicerMock("")
	notifMock := NotificationMock{}
	rw := getResponseWriterMock()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/ping", nil)
	expected, _ := json.Marshal(Response{Status: "OK"})

	srv := NewServe(servicerMock, notifMock)
	srv.PingHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
	rw.AssertCalled(s.T(), "Write", []byte(expected))
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

func (m *ResponseWriterMock) GetLastMethodCall(methodName string) *mock.Call {
	for _, call := range m.Calls {
		if call.Method == methodName {
			return &call
		}
	}
	return nil
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

func (m *ServicerMock) GetServices() (*[]service.SwarmService, error) {
	args := m.Called()
	s := args.Get(0).([]service.SwarmService)
	return &s, args.Error(1)
}

func (m *ServicerMock) GetNewServices(services *[]service.SwarmService) (*[]service.SwarmService, error) {
	args := m.Called()
	return args.Get(0).(*[]service.SwarmService), args.Error(1)
}

func (m *ServicerMock) GetRemovedServices(services *[]service.SwarmService) *[]string {
	args := m.Called(services)
	return args.Get(0).(*[]string)
}

func (m *ServicerMock) GetServicesParameters(services *[]service.SwarmService) *[]map[string]string {
	args := m.Called(services)
	return args.Get(0).(*[]map[string]string)
}

func getServicerMock(skipMethod string) *ServicerMock {
	mockObj := new(ServicerMock)
	if !strings.EqualFold("GetServices", skipMethod) {
		mockObj.On("GetServices").Return([]service.SwarmService{}, nil)
	}
	if !strings.EqualFold("GetNewServices", skipMethod) {
		mockObj.On("GetNewServices", mock.Anything).Return([]service.SwarmService{}, nil)
	}
	if !strings.EqualFold("GetRemovedServices", skipMethod) {
		mockObj.On("GetRemovedServices", mock.Anything).Return(&[]string{})
	}
	if !strings.EqualFold("GetServicesParameters", skipMethod) {
		mockObj.On("GetServicesParameters", mock.Anything).Return(&[]map[string]string{})
	}
	return mockObj
}

type NotificationMock struct {
	ServicesCreateMock func(services *[]service.SwarmService, retries, interval int) error
	ServicesRemoveMock func(remove *[]string, retries, interval int) error
}

func (m NotificationMock) ServicesCreate(services *[]service.SwarmService, retries, interval int) error {
	return m.ServicesCreateMock(services, retries, interval)
}

func (m NotificationMock) ServicesRemove(remove *[]string, retries, interval int) error {
	return m.ServicesRemoveMock(remove, retries, interval)
}
