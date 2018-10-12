package service

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/suite"
)

type SwarmServiceListenerTestSuite struct {
	suite.Suite
	ServiceName  string
	DockerClient *client.Client
	Logger       *log.Logger
}

func TestSwarmServiceListenerTestSuite(t *testing.T) {
	suite.Run(t, new(SwarmServiceListenerTestSuite))
}

func (s *SwarmServiceListenerTestSuite) SetupSuite() {
	s.ServiceName = "my-service"
	client, err := NewDockerClientFromEnv()
	s.Require().NoError(err)
	s.DockerClient = client
	s.Logger = log.New(os.Stdout, "", 0)
}

func (s *SwarmServiceListenerTestSuite) Test_ListenForServiceEvents_CreateService() {
	snl := NewSwarmServiceListener(s.DockerClient, s.Logger)

	// Listen for events
	eventChan := make(chan Event)
	snl.ListenForServiceEvents(eventChan)

	createTestService("util-1", []string{}, false, "", "")
	defer func() {
		removeTestService("util-1")
	}()

	time.Sleep(time.Second)
	utilID, err := getServiceID("util-1")
	s.Require().NoError(err)

	event, err := s.waitForServiceEvent(eventChan)
	s.Require().NoError(err)

	s.Equal(EventTypeCreate, event.Type)
	s.Equal(utilID, event.ID)
	s.True(event.ConsultCache)
}

func (s *SwarmServiceListenerTestSuite) Test_ListenForServiceEvents_UpdateService() {
	snl := NewSwarmServiceListener(s.DockerClient, s.Logger)

	createTestService("util-1", []string{}, false, "", "")
	defer func() {
		removeTestService("util-1")
	}()

	time.Sleep(time.Second)
	utilID, err := getServiceID("util-1")
	s.Require().NoError(err)

	// Listen for events
	eventChan := make(chan Event)
	snl.ListenForServiceEvents(eventChan)

	// Update label
	addLabelToService("util-1", "hello=world")

	event, err := s.waitForServiceEvent(eventChan)
	s.Require().NoError(err)

	s.Equal(EventTypeCreate, event.Type)
	s.Equal(utilID, event.ID)
	s.True(event.ConsultCache)

	// Remove label
	removeLabelFromService("util-1", "hello")

	event, err = s.waitForServiceEvent(eventChan)
	s.Require().NoError(err)

	s.Equal(EventTypeCreate, event.Type)
	s.Equal(utilID, event.ID)
	s.True(event.ConsultCache)
}

func (s *SwarmServiceListenerTestSuite) Test_ListenForServiceEvents_RemoveService() {
	snl := NewSwarmServiceListener(s.DockerClient, s.Logger)

	createTestService("util-1", []string{}, false, "", "")
	defer func() {
		removeTestService("util-1")
	}()

	time.Sleep(time.Second)
	utilID, err := getServiceID("util-1")
	s.Require().NoError(err)

	// Listen for events
	eventChan := make(chan Event)
	snl.ListenForServiceEvents(eventChan)

	// Remove service
	removeTestService("util-1")

	event, err := s.waitForServiceEvent(eventChan)
	s.Require().NoError(err)

	s.Equal(EventTypeRemove, event.Type)
	s.Equal(utilID, event.ID)
	s.True(event.ConsultCache)
}

func (s *SwarmServiceListenerTestSuite) waitForServiceEvent(events <-chan Event) (*Event, error) {
	timeOut := time.NewTimer(time.Second * 5).C
	for {
		select {
		case event := <-events:
			return &event, nil
		case <-timeOut:
			return nil, fmt.Errorf("Timeout")
		}
	}
}
