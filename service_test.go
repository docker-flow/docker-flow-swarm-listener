package main

import (
	"fmt"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/suite"
	"net/http"
	"testing"
	"github.com/docker/docker/api/types/swarm"
	"net/http/httptest"
)

type ServicesTestSuite struct {
	suite.Suite
	serviceName string
//	Server *httptest.Server
}

func TestServicesUnitTestSuite(t *testing.T) {
	s := new(ServicesTestSuite)
	s.serviceName = "my-service"
	suite.Run(t, s)
}

func (s *ServicesTestSuite) SetupTest() {
//	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		actualPath := r.URL.Path
//		if r.Method == "GET" {
//			switch actualPath {
//			case "/v1/docker-flow-proxy/reconfigure":
//				w.WriteHeader(http.StatusOK)
//				w.Header().Set("Content-Type", "application/json")
//			default:
//				w.WriteHeader(http.StatusNotFound)
//			}
//		}
//	}))
//	defer func() { s.Server.Close() }()
}

// GetServices

func (s *ServicesTestSuite) Test_GetServices_ReturnsServices() {
	services := NewServices()

	actual, _ := services.GetServices("unix:///var/run/docker.sock")

	s.Equal(2, len(actual))
	index := 0
	if actual[1].Spec.Name == "util-1" {
		index = 1
	}
	s.Equal("util-1", actual[index].Spec.Name)
	s.Equal("/demo", actual[index].Spec.Labels["DF_SERVICE_PATH"])
}

func (s *ServicesTestSuite) Test_GetServices_ReturnsError_WhenNewClientFails() {
	dcOrig := dockerClient
	defer func() { dockerClient = dcOrig }()
	dockerClient = func(host string, version string, httpClient *http.Client, httpHeaders map[string]string) (*client.Client, error) {
		return &client.Client{}, fmt.Errorf("This is an error")
	}
	services := NewServices()
	_, err := services.GetServices("unix:///var/run/docker.sock")
	s.Error(err)
}

func (s *ServicesTestSuite) Test_GetServices_ReturnsError_WhenServiceListFails() {
	services := NewServices()
	_, err := services.GetServices("unix:///this/socket/does/not/exist")
	s.Error(err)
}

// NotifyServices

func (s *ServicesTestSuite) Test_NotifyServices_SendsRequests() {
	labels := make(map[string]string)
	labels["DF_NOTIFY"] = "true"
	labels["DF_key1"] = "value1"
	labels["VAR_WITHOUT_DF_PREFIX"] = "something"

	s.verifyNotifyService(labels, true, fmt.Sprintf("serviceName=%s&key1=value1", s.serviceName))
}

func (s *ServicesTestSuite) Test_NotifyServices_DoesNotSendRequest_WhenDfNotifyIsNotDefined() {
	labels := make(map[string]string)
	labels["DF_key1"] = "value1"

	s.verifyNotifyService(labels, false, "")
}

func (s *ServicesTestSuite) Test_NotifyServices_ReturnsError_WhenHttpStatusIsNot200() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	labels := make(map[string]string)
	labels["DF_NOTIFY"] = "true"

	services := NewServices()
	err := services.NotifyServices(s.getSwarmServices(labels), httpSrv.URL)

	s.Error(err)
}

func (s *ServicesTestSuite) Test_NotifyServices_ReturnsError_WhenHttpRequestReturnsError() {
//	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusNotFound)
//	}))
	labels := make(map[string]string)
	labels["DF_NOTIFY"] = "true"

	services := NewServices()
	err := services.NotifyServices(s.getSwarmServices(labels), "http://this-does-not-exist")

	s.Error(err)
}

// Util

func (s *ServicesTestSuite) verifyNotifyService(labels map[string]string, expectSent bool, expectQuery string) {
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

	services := NewServices()
	err := services.NotifyServices(s.getSwarmServices(labels), url)

	s.NoError(err)
	s.Equal(expectSent, actualSent)
	if expectSent {
		s.Equal(expectQuery, actualQuery)
	}
}

func (s *ServicesTestSuite) getSwarmServices(labels map[string]string) []swarm.Service {
	ann := swarm.Annotations{
		Name: s.serviceName,
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

