package main

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"fmt"
	"net/http"
	"github.com/stretchr/testify/mock"
	"github.com/docker/docker/api/types/swarm"
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
	expected := fmt.Sprintf("localhost:8080")
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

	srv := NewServe(getServicerMock(""))
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

	srv := NewServe(getServicerMock(""))
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

	srv := NewServe(getServicerMock(""))
	srv.ServeHTTP(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 404)
}

func (s *ServerTestSuite) Test_ServeHTTP_InvokesNotifyServicesCreate_WhenUrlIsNotifyservices() {
	mockObj := getServicerMock("GetServices")
	service1 := swarm.Service{
		ID: "my-service-id-1",
	}
	services := []swarm.Service{service1}
	mockObj.On("GetServices").Return(services, nil)
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	rw := getResponseWriterMock()

	srv := NewServe(mockObj)
	srv.ServeHTTP(rw, req)

	mockObj.AssertCalled(s.T(), "NotifyServicesCreate", services, 3, 5)
}

// NewServe

func (s *ServerTestSuite) Test_NewServe_SetsService() {
	service := NewServiceFromEnv()
	serve := NewServe(service)

	s.Equal(service, serve.Service)
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