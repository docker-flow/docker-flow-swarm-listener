package main

import (
	"fmt"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/suite"
	"net/http"
	"testing"
)

type ServicesTestSuite struct {
	suite.Suite
}

func TestServicesUnitTestSuite(t *testing.T) {
	s := new(ServicesTestSuite)
	suite.Run(t, s)
}

func (s *ServicesTestSuite) SetupTest() {
}

func (s *ServicesTestSuite) Test_GetServices_ReturnsServices() {
	services := NewServices()
	actual, _ := services.GetServices("unix:///var/run/docker.sock")
	s.Equal(1, len(actual))
	s.Equal("util", actual[0].Spec.Name)
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

func (s *ServicesTestSuite) Test_GetServices_Xxx() {
	services := NewServices()
	_, err := services.GetServices("unix:///this/socker/does/not/exist")
	s.Error(err)
}
