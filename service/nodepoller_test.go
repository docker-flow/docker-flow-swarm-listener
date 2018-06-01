package service

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type NodePollerTestSuite struct {
	suite.Suite
	NodeClientMock *nodeInspectorMock
	NodeCacheMock  *nodeCacherMock

	NodePoller *NodePoller
	Logger     *log.Logger
	LogBytes   *bytes.Buffer
}

func TestNodePollerUnitTestSuite(t *testing.T) {
	suite.Run(t, new(NodePollerTestSuite))
}

func (s *NodePollerTestSuite) SetupTest() {
	s.NodeClientMock = new(nodeInspectorMock)
	s.NodeCacheMock = new(nodeCacherMock)

	s.LogBytes = new(bytes.Buffer)
	s.Logger = log.New(s.LogBytes, "", 0)

	s.NodePoller = NewNodePoller(
		s.NodeClientMock,
		s.NodeCacheMock,
		1,
		MinifyNode,
		s.Logger,
	)
}

func (s *NodePollerTestSuite) Test_Run_NoCache() {

	expNodes := []swarm.Node{
		{ID: "nodeID1"}, {ID: "nodeID2"},
	}
	keys := map[string]struct{}{}
	miniNode1 := NodeMini{ID: "nodeID1", EngineLabels: map[string]string{}, NodeLabels: map[string]string{}}
	miniNode2 := NodeMini{ID: "nodeID2", EngineLabels: map[string]string{}, NodeLabels: map[string]string{}}

	eventChan := make(chan Event)

	s.NodeClientMock.
		On("NodeList", mock.AnythingOfType("*context.emptyCtx")).Return(expNodes, nil)

	s.NodeCacheMock.
		On("Keys").Return(keys).
		On("IsNewOrUpdated", miniNode1).Return(true).
		On("IsNewOrUpdated", miniNode2).Return(true)

	go s.NodePoller.Run(eventChan)

	timeout := time.NewTimer(time.Second * 5).C
	eventsNum := 0

	for {
		if eventsNum == 2 {
			break
		}
		select {
		case event := <-eventChan:
			s.Require().Equal(EventTypeCreate, event.Type)
			eventsNum++
		case <-timeout:
			s.FailNow("Timeout")
		}
	}

	s.Equal(2, eventsNum)
	s.NodeClientMock.AssertExpectations(s.T())
	s.NodeCacheMock.AssertExpectations(s.T())
}

func (s *NodePollerTestSuite) Test_Run_HalfInCache() {
	expNodes := []swarm.Node{
		{ID: "nodeID1"}, {ID: "nodeID2"},
	}
	miniNode1 := NodeMini{ID: "nodeID1", EngineLabels: map[string]string{}, NodeLabels: map[string]string{}}
	miniNode2 := NodeMini{ID: "nodeID2", EngineLabels: map[string]string{}, NodeLabels: map[string]string{}}

	keys := map[string]struct{}{}
	keys["nodeID1"] = struct{}{}

	eventChan := make(chan Event)

	s.NodeClientMock.
		On("NodeList", mock.AnythingOfType("*context.emptyCtx")).Return(expNodes, nil)

	s.NodeCacheMock.
		On("Keys").Return(keys).
		On("IsNewOrUpdated", miniNode1).Return(false).
		On("IsNewOrUpdated", miniNode2).Return(true)

	go s.NodePoller.Run(eventChan)

	timeout := time.NewTimer(time.Second * 5).C
	var eventCreate *Event
	eventsNum := 0

	for {
		if eventsNum == 1 {
			break
		}
		select {
		case event := <-eventChan:
			if event.ID == "nodeID2" {
				eventCreate = &event
			}
			eventsNum++
		case <-timeout:
			s.FailNow("Timeout")
		}
	}

	s.Equal(1, eventsNum)
	s.Require().NotNil(eventCreate)

	s.Equal("nodeID2", eventCreate.ID)
	s.NodeClientMock.AssertExpectations(s.T())
	s.NodeCacheMock.AssertExpectations(s.T())
}

func (s *NodePollerTestSuite) Test_Run_MoreInCache() {
	expNodes := []swarm.Node{
		{ID: "nodeID1"}, {ID: "nodeID2"},
	}
	miniNode1 := NodeMini{ID: "nodeID1", EngineLabels: map[string]string{}, NodeLabels: map[string]string{}}
	miniNode2 := NodeMini{ID: "nodeID2", EngineLabels: map[string]string{}, NodeLabels: map[string]string{}}

	keys := map[string]struct{}{}
	keys["nodeID1"] = struct{}{}
	keys["nodeID2"] = struct{}{}
	keys["nodeID3"] = struct{}{}

	eventChan := make(chan Event)

	s.NodeClientMock.
		On("NodeList", mock.AnythingOfType("*context.emptyCtx")).Return(expNodes, nil)

	s.NodeCacheMock.
		On("Keys").Return(keys).
		On("IsNewOrUpdated", miniNode1).Return(true).
		On("IsNewOrUpdated", miniNode2).Return(false)

	go s.NodePoller.Run(eventChan)

	timeout := time.NewTimer(time.Second * 5).C
	var eventCreate *Event
	var eventRemove *Event
	eventsNum := 0

	for {
		if eventsNum == 2 {
			break
		}
		select {
		case event := <-eventChan:
			if event.ID == "nodeID1" {
				eventCreate = &event
			} else if event.ID == "nodeID3" {
				eventRemove = &event
			}
			eventsNum++
		case <-timeout:
			s.FailNow("Timeout")
		}
	}

	s.Equal(2, eventsNum)
	s.Require().NotNil(eventCreate)
	s.Require().NotNil(eventRemove)

	s.Equal("nodeID1", eventCreate.ID)
	s.Equal("nodeID3", eventRemove.ID)
	s.NodeClientMock.AssertExpectations(s.T())
	s.NodeCacheMock.AssertExpectations(s.T())

}
