package service

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/suite"
)

type ServiceTestSuite struct {
	suite.Suite
	serviceName string
}

func TestServiceUnitTestSuite(t *testing.T) {
	s := new(ServiceTestSuite)
	s.serviceName = "my-service"

	logPrintfOrig := logPrintf
	defer func() {
		logPrintf = logPrintfOrig
		os.Unsetenv("DF_NOTIFY_LABEL")
	}()
	logPrintf = func(format string, v ...interface{}) {}
	os.Setenv("DF_NOTIFY_LABEL", "com.df.notify")

	createTestServices()
	suite.Run(t, s)
	removeTestServices()
}

// GetServices

func (s *ServiceTestSuite) Test_GetServices_ReturnsServices() {
	service := NewService("unix:///var/run/docker.sock")

	services, _ := service.GetServices()
	actual := *services

	s.Equal(2, len(actual))
	s.Equal("/demo", actual[0].Spec.Labels["com.df.servicePath"])
	s.Equal("true", actual[0].Spec.Labels["com.df.distribute"])
}

func (s *ServiceTestSuite) Test_GetServices_ReturnsError_WhenServiceListFails() {
	services := NewService("unix:///this/socket/does/not/exist")

	_, err := services.GetServices()

	s.Error(err)
}

// GetNewServices

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsAllServices_WhenExecutedForTheFirstTime() {
	service := NewService("unix:///var/run/docker.sock")
	service.ServiceLastUpdatedAt = time.Time{}
	services, _ := service.GetServices()

	actual, _ := service.GetNewServices(services)

	s.Equal(2, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_ReturnsOnlyNewServices() {
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(0, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsServices() {
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	service.GetNewServices(services)

	s.Equal(2, len(CachedServices))
	s.Contains(CachedServices, "util-1")
	s.Contains(CachedServices, "util-3")
}

func (s *ServiceTestSuite) Test_GetNewServices_DoesNotAddServices_WhenReplicasAreZero() {
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()
	for _, s := range *services {
		if s.Spec.Name == "util-1" {
			replicas := uint64(0)
			s.Spec.Mode.Replicated.Replicas = &replicas
		}
	}

	service.GetNewServices(services)

	s.NotContains(CachedServices, "util-1")
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsServices_WhenModeIsGlobal() {
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	service.GetNewServices(services)

	s.Equal(2, len(CachedServices))
	s.Contains(CachedServices, "util-3")
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsUpdatedServices_WhenLabelIsAdded() {
	defer func() {
		exec.Command("docker", "service", "update", "--label-rm", "com.df.something", "util-1").Output()
	}()
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--label-add", "com.df.something=else", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(1, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_DoesNotAddUpdatedServices_WhenComDfLabelsDidNotChange() {
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--label-add", "something=else", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(0, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsUpdatedServices_WhenLabelIsRemoved() {
	exec.Command("docker", "service", "update", "--label-add", "com.df.something=else", "util-1").Output()
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--label-rm", "com.df.something", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(1, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsUpdatedServices_WhenLabelIsUpdated() {
	defer func() {
		exec.Command("docker", "service", "update", "--label-rm", "com.df.something", "util-1").Output()
	}()
	exec.Command("docker", "service", "update", "--label-add", "com.df.something=else", "util-1").Output()
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--label-add", "com.df.something=little-piggy", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(1, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsUpdatedServices_WhenReplicasAreUpdated() {
	defer func() {
		exec.Command("docker", "service", "update", "--label-rm", "com.df.something", "--replicas", "1", "util-1").Output()
	}()
	exec.Command("docker", "service", "update", "--replicas", "1", "util-1").Output()
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--replicas", "2", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(1, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_DoesNotAddServices_WhenReplicasAreSetTo0() {
	defer func() {
		exec.Command("docker", "service", "update", "--label-rm", "com.df.something", "--replicas", "1", "util-1").Output()
	}()
	exec.Command("docker", "service", "update", "--replicas", "1", "util-1").Output()
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--replicas", "0", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Equal(0, len(*actual))
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsUpdatedServices_WhenReplicasAreUpdated_NodeInfo() {
	defer func() {
		exec.Command("docker", "service", "update",
			"--label-rm", "com.df.something",
			"--label-rm", "com.df.scrapeNetwork",
			"--replicas", "1", "util-1").Output()
		os.Unsetenv("DF_INCLUDE_NODE_IP_INFO")
	}()
	os.Setenv("DF_INCLUDE_NODE_IP_INFO", "true")

	exec.Command("docker", "service", "update",
		"--label-add", "com.df.scrapeNetwork=util-network",
		"--replicas", "1", "util-1").Output()
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--replicas", "2", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Require().Len(*actual, 1)
	actualService := (*actual)[0]

	s.Equal(2, actualService.NodeInfo.Cardinality())
}

func (s *ServiceTestSuite) Test_GetNewServices_AddsUpdatedServices_WhenReplicasAreUpdated_NodeInfo_IncorrectNetworkLabel() {
	defer func() {
		exec.Command("docker", "service", "update",
			"--label-rm", "com.df.something",
			"--label-rm", "com.df.scrapeNetwork",
			"--replicas", "1", "util-1").Output()
		os.Unsetenv("DF_INCLUDE_NODE_IP_INFO")
	}()
	os.Setenv("DF_INCLUDE_NODE_IP_INFO", "true")

	exec.Command("docker", "service", "update",
		"--label-add", "com.df.scrapeNetwork=bad",
		"--replicas", "1", "util-1").Output()
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()

	exec.Command("docker", "service", "update", "--replicas", "2", "util-1").Output()
	service.GetNewServices(services)
	services, _ = service.GetServices()
	actual, _ := service.GetNewServices(services)

	s.Require().Len(*actual, 1)
	actualService := (*actual)[0]

	s.Equal(0, actualService.NodeInfo.Cardinality())
}

// GetRemovedServices

func (s *ServiceTestSuite) Test_GetRemovedServices_ReturnsNamesOfRemovedServices() {
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()
	CachedServices["removed-service-1"] = SwarmService{}
	CachedServices["removed-service-2"] = SwarmService{}

	actual := service.GetRemovedServices(services)

	s.Equal(2, len(*actual))
	s.Contains(*actual, "removed-service-1")
	s.Contains(*actual, "removed-service-2")
}

func (s *ServiceTestSuite) Test_GetRemovedServices_AddsServicesWithZeroReplicas() {
	service := NewService("unix:///var/run/docker.sock")
	services, _ := service.GetServices()
	CachedServices["util-1"] = SwarmService{}
	for _, s := range *services {
		if s.Spec.Mode.Replicated != nil {
			replicas := uint64(0)
			s.Spec.Mode.Replicated.Replicas = &replicas
		}
	}
	actual := service.GetRemovedServices(services)

	s.Equal(1, len(*actual))
	s.Contains(*actual, "util-1")
}

// GetServicesParameters

func (s *ServiceTestSuite) Test_GetRemovedServices_GetServicesParameters() {
	service := NewService("unix:///var/run/docker.sock")
	replicas := uint64(1)
	mode := swarm.ServiceMode{
		Replicated: &swarm.ReplicatedService{Replicas: &replicas},
	}
	srv := swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "demo",
				Labels: map[string]string{
					"com.df.notify":      "true",
					"com.df.servicePath": "/demo",
					"com.df.distribute":  "true",
				},
			},
			Mode: mode,
		},
	}
	srvs := []SwarmService{{srv, nil}}
	paramsList := service.GetServicesParameters(&srvs)
	expected := []map[string]string{
		{
			"serviceName": "demo",
			"servicePath": "/demo",
			"distribute":  "true",
			"replicas":    "1",
		},
	}
	s.Equal(&expected, paramsList)
}

func (s *ServiceTestSuite) Test_GetServiceParametersWithNodeInfo() {
	service := NewService("unix:///var/run/docker.sock")
	replicas := uint64(1)
	mode := swarm.ServiceMode{
		Replicated: &swarm.ReplicatedService{Replicas: &replicas},
	}
	srv := swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "demo",
				Labels: map[string]string{
					"com.df.notify":      "true",
					"com.df.servicePath": "/demo",
					"com.df.distribute":  "true",
				},
			},
			Mode: mode,
		},
	}
	nodeInfo := NodeIPSet{}
	nodeInfo.Add("node-1", "10.0.1.1")
	nodeInfo.Add("node-1", "10.0.1.2")
	srvs := []SwarmService{{srv, &nodeInfo}}
	paramsList := service.GetServicesParameters(&srvs)
	s.Require().Len(*paramsList, 1)

	params := (*paramsList)[0]
	expected := map[string]string{
		"serviceName": "demo",
		"servicePath": "/demo",
		"distribute":  "true",
		"replicas":    "1",
	}

	for k, v := range expected {
		s.Equal(v, params[k])
	}

	nodeInfoStr := params["nodeInfo"]
	paramNodeInfo := NodeIPSet{}
	err := json.Unmarshal([]byte(nodeInfoStr), &paramNodeInfo)
	s.Require().NoError(err)

	s.True(nodeInfo.Equal(paramNodeInfo))
}

func (s *ServiceTestSuite) Test_GetRemovedServices_IgnoresThoseScaledToZero() {
	service := NewService("unix:///var/run/docker.sock")
	replicas := uint64(0)
	mode := swarm.ServiceMode{
		Replicated: &swarm.ReplicatedService{Replicas: &replicas},
	}
	srv := swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "demo",
				Labels: map[string]string{
					"com.df.notify":      "true",
					"com.df.servicePath": "/demo",
					"com.df.distribute":  "true",
				},
			},
			Mode: mode,
		},
	}
	srvs := []SwarmService{{srv, nil}}
	paramsList := service.GetServicesParameters(&srvs)
	expected := []map[string]string{}
	s.Equal(&expected, paramsList)
}

// NewService

func (s *ServiceTestSuite) Test_NewService_SetsHost() {
	expected := "this-is-a-host"

	service := NewService(expected)

	s.Equal(expected, service.Host)
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

// Util

func createTestServices() {
	createTestNetwork("util-network")
	createTestService("util-1", []string{"com.df.notify=true", "com.df.servicePath=/demo", "com.df.distribute=true"}, "", "util-network")
	createTestService("util-2", []string{}, "", "util-network")
	createTestService("util-3", []string{"com.df.notify=true", "com.df.servicePath=/demo", "com.df.distribute=true"}, "global", "util-network")
}

func createTestNetwork(name string) {
	args := []string{"network", "create", "-d", "overlay", name}
	exec.Command("docker", args...).Output()
}

func createTestService(name string, labels []string, mode string, network string) {
	args := []string{"service", "create", "--name", name}
	for _, v := range labels {
		args = append(args, "-l", v)
	}
	if len(mode) > 0 {
		args = append(args, "--mode", "global")
	}
	if len(network) > 0 {
		args = append(args, "--network", network)
	}
	args = append(args, "alpine", "sleep", "1000000000")
	exec.Command("docker", args...).Output()
}

func removeTestServices() {
	removeTestService("util-1")
	removeTestService("util-2")
	removeTestService("util-3")
	removeTestNetwork("util-network")
}

func removeTestService(name string) {
	exec.Command("docker", "service", "rm", name).Output()
}

func removeTestNetwork(name string) {
	exec.Command("docker", "network", "rm", name).Output()
}
