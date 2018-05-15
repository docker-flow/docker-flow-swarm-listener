package service

import (
	"bytes"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/suite"
)

type EventListenerNodeTestSuite struct {
	suite.Suite
	DockerClient   *client.Client
	Logger         *log.Logger
	LogBytes       *bytes.Buffer
	NetworkName    string
	Node0          string
	Node0JoinToken string
}

func TestEventListenerNodeTestSuite(t *testing.T) {
	suite.Run(t, new(EventListenerNodeTestSuite))
}

func (s *EventListenerNodeTestSuite) SetupSuite() {
	s.LogBytes = new(bytes.Buffer)
	s.Logger = log.New(s.LogBytes, "", 0)

	// Assumes running test with docker-compose.yml
	network, err := getNetworkNameWithSuffix("dfsl_network")
	s.Require().NoError(err)
	s.NetworkName = network
	s.Node0 = "node0"

	createNode(s.Node0, s.NetworkName)
	time.Sleep(time.Second)
	initSwarm(s.Node0)
	time.Sleep(time.Second)

	s.Node0JoinToken = getWorkerToken(s.Node0)
	time.Sleep(time.Second)

	client, err := newTestNodeDockerClient(s.Node0)
	s.Require().NoError(err)
	s.DockerClient = client

}

func (s *EventListenerNodeTestSuite) TearDownSuite() {
	destroyNode(s.Node0)
}

func (s *EventListenerNodeTestSuite) Test_ListenForNodeEvents_NodeCreate() {

	enl := NewNodeListener(s.DockerClient, s.Logger)

	// Listen for events
	eventChan := make(chan Event)
	enl.ListenForNodeEvents(eventChan)

	// Create node1
	createNode("node1", s.NetworkName)
	defer func() {
		destroyNode("node1")
	}()

	time.Sleep(time.Second)
	joinSwarm("node1", s.Node0, s.Node0JoinToken)

	// Wait for events
	event, err := s.waitForEvent(eventChan)
	s.Require().NoError(err)
	s.True(event.UseCache)

	node1ID, err := getNodeID("node1", "node0")
	s.Require().NoError(err)

	s.Equal(node1ID, event.ID)
	s.Equal(EventTypeCreate, event.Type)
}

// This test is not consistent
// func (s *EventListenerNodeTestSuite) Test_ListenForNodeEvents_NodeRemove() {

// 	enl := NewNodeListener(s.DockerClient, s.Logger)

// 	// Create node1 and joing swarm
// 	createNode("node1", s.NetworkName)
// 	defer func() {
// 		destroyNode("node1")
// 	}()
// 	joinSwarm("node1", s.Node0, s.Node0JoinToken)

// 	node1ID, err := getNodeID("node1", s.Node0)
// 	s.Require().NoError(err)

// 	// Listen for events
// 	eventChan := make(chan Event)
// 	enl.ListenForNodeEvents(eventChan)

// 	//Remove node1
// 	removeNodeFromSwarm("node1", s.Node0)

// 	// Wait for events
// 	event, err := s.waitForEvent(eventChan)
// 	s.Require().NoError(err)

// 	s.Equal(node1ID, event.ID)
// 	s.Equal(EventTypeRemove, event.Type)
// }

func (s *EventListenerNodeTestSuite) Test_ListenForNodeEvents_NodeUpdateLabel() {
	// Create one node
	enl := NewNodeListener(s.DockerClient, s.Logger)

	// Listen for events
	eventChan := make(chan Event)
	enl.ListenForNodeEvents(eventChan)

	// addLabelToNode
	addLabelToNode(s.Node0, "cats=flay", s.Node0)

	// Wait for events
	event, err := s.waitForEvent(eventChan)
	s.Require().NoError(err)
	s.True(event.UseCache)

	node0ID, err := getNodeID(s.Node0, s.Node0)
	s.Require().NoError(err)

	s.Equal(node0ID, event.ID)
	s.Equal(EventTypeCreate, event.Type)

	// removeLabelFromNode
	removeLabelFromNode(s.Node0, "cats", s.Node0)

	// Wait for events
	event, err = s.waitForEvent(eventChan)
	s.Require().NoError(err)
	s.True(event.UseCache)

	s.Equal(node0ID, event.ID)
	s.Equal(EventTypeCreate, event.Type)
}

func (s *EventListenerNodeTestSuite) waitForEvent(events <-chan Event) (*Event, error) {
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
