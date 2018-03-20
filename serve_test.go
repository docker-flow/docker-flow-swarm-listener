package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	Log    *log.Logger
	RWMock *ResponseWriterMock
	SLMock *SwarmListeningMock
}

func TestServerUnitTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) SetupTest() {
	s.Log = log.New(os.Stdout, "", 0)

	s.RWMock = new(ResponseWriterMock)
	s.RWMock.On("Header").Return(nil)
	s.RWMock.On("Write", mock.Anything).Return(0, nil)
	s.RWMock.On("WriteHeader", mock.Anything)
	s.SLMock = new(SwarmListeningMock)
}

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

func (s *ServerTestSuite) Test_NotifyServices_ReturnsStatus200() {
	s.SLMock.On("NotifyServices", false).Return()

	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	expected, _ := json.Marshal(Response{Status: "OK"})

	srv := NewServe(s.SLMock, s.Log)
	srv.NotifyServices(s.RWMock, req)

	s.RWMock.AssertCalled(s.T(), "WriteHeader", 200)
	s.RWMock.AssertCalled(s.T(), "Write", []byte(expected))
	s.SLMock.AssertExpectations(s.T())
}

func (s *ServerTestSuite) Test_NotifyServices_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentTypeOrig := httpWriterSetContentType
	defer func() { httpWriterSetContentType = httpWriterSetContentTypeOrig }()
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/notify-services", nil)
	s.SLMock.On("NotifyServices", false).Return()

	srv := NewServe(s.SLMock, s.Log)
	srv.NotifyServices(s.RWMock, req)

	s.Equal("application/json", actual)
	s.SLMock.AssertExpectations(s.T())
}

// GetServices

func (s *ServerTestSuite) Test_GetServices_ReturnsServices() {
	mapParam := []map[string]string{
		{
			"serviceName": "demo",
			"notify":      "true",
			"servicePath": "/demo",
			"distribute":  "true",
		},
	}
	s.SLMock.On("GetServicesParameters", mock.Anything).Return(mapParam, nil)
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/get-services", nil)
	srv := NewServe(s.SLMock, s.Log)
	srv.GetServices(s.RWMock, req)

	call := s.RWMock.GetLastMethodCall("Write")
	value, _ := call.Arguments.Get(0).([]byte)
	rsp := []map[string]string{}
	json.Unmarshal(value, &rsp)
	s.Equal(mapParam, rsp)
}

// GetNodes

func (s *ServerTestSuite) Test_GetNodes_ReturnNodes() {
	mapParam := []map[string]string{
		{
			"id":           "node1",
			"hostname":     "node1hostname",
			"address":      "10.0.0.1",
			"versionIndex": "24",
			"state":        "ready",
			"role":         "worker",
			"availability": "active",
		},
	}
	s.SLMock.On("GetNodesParameters", mock.Anything).Return(mapParam, nil)
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/get-nodes", nil)
	srv := NewServe(s.SLMock, s.Log)
	srv.GetNodes(s.RWMock, req)
	call := s.RWMock.GetLastMethodCall("Write")
	value, _ := call.Arguments.Get(0).([]byte)
	rsp := []map[string]string{}
	json.Unmarshal(value, &rsp)
	s.Equal(mapParam, rsp)
}

// PingHandler

func (s *ServerTestSuite) Test_PingHandler_ReturnsStatus200() {
	actual := ""
	httpWriterSetContentTypeOrig := httpWriterSetContentType
	defer func() { httpWriterSetContentType = httpWriterSetContentTypeOrig }()
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-swarm-listener/ping", nil)
	expected, _ := json.Marshal(Response{Status: "OK"})

	srv := NewServe(s.SLMock, s.Log)
	srv.PingHandler(s.RWMock, req)

	s.Equal("application/json", actual)
	s.RWMock.AssertCalled(s.T(), "WriteHeader", 200)
	s.RWMock.AssertCalled(s.T(), "Write", []byte(expected))
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

type SwarmListeningMock struct {
	mock.Mock
}

func (m *SwarmListeningMock) Run() {
	m.Called()
}
func (m *SwarmListeningMock) NotifyServices(ignoreCache bool) {
	m.Called(ignoreCache)
}
func (m *SwarmListeningMock) NotifyNodes(ignoreCache bool) {
	m.Called(ignoreCache)
}
func (m *SwarmListeningMock) GetServicesParameters(ctx context.Context) ([]map[string]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]map[string]string), args.Error(1)
}
func (m *SwarmListeningMock) GetNodesParameters(ctx context.Context) ([]map[string]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]map[string]string), args.Error(1)
}
