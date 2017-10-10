package service

import (
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/suite"
	"os"
	"os/exec"
	"testing"
	"time"
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
		},
	}
	srvs := []SwarmService{{srv}}
	paramsList := service.GetServicesParameters(&srvs)
	expected := []map[string]string{
		{"serviceName": "demo",
			"servicePath": "/demo",
			"distribute":  "true"},
	}
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
	createTestService("util-1", []string{"com.df.notify=true", "com.df.servicePath=/demo", "com.df.distribute=true"}, "")
	createTestService("util-2", []string{}, "")
	createTestService("util-3", []string{"com.df.notify=true", "com.df.servicePath=/demo", "com.df.distribute=true"}, "global")
}

func createTestService(name string, labels []string, mode string) {
	args := []string{"service", "create", "--name", name}
	for _, v := range labels {
		args = append(args, "-l", v)
	}
	if len(mode) > 0 {
		args = append(args, "--mode", "global")
	}
	args = append(args, "alpine", "sleep", "1000000000")
	exec.Command("docker", args...).Output()
}

func removeTestServices() {
	removeTestService("util-1")
	removeTestService("util-2")
	removeTestService("util-3")
}

func removeTestService(name string) {
	exec.Command("docker", "service", "rm", name).Output()
}
