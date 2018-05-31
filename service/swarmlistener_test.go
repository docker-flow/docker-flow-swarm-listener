package service

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type SwarmListenerTestSuite struct {
	suite.Suite
	SSListenerMock *swarmServiceListeningMock
	SSClientMock   *swarmServiceInspector
	SSCacheMock    *swarmServiceCacherMock

	NodeListeningMock *nodeListeningMock
	NodeClientMock    *nodeInspectorMock
	NodeCacheMock     *nodeCacherMock

	SSPoller *swarmServicePollingMock

	NotifyDistributorMock *notifyDistributorMock

	SwarmListener *SwarmListener
	Logger        *log.Logger
	LogBytes      *bytes.Buffer
}

func TestSwarmListenerUnitTestSuite(t *testing.T) {
	suite.Run(t, new(SwarmListenerTestSuite))
}

func (s *SwarmListenerTestSuite) SetupTest() {

	s.SSListenerMock = new(swarmServiceListeningMock)
	s.SSClientMock = new(swarmServiceInspector)
	s.SSCacheMock = new(swarmServiceCacherMock)
	s.NodeListeningMock = new(nodeListeningMock)
	s.NodeClientMock = new(nodeInspectorMock)
	s.NodeCacheMock = new(nodeCacherMock)

	s.SSPoller = new(swarmServicePollingMock)

	s.NotifyDistributorMock = new(notifyDistributorMock)
	s.LogBytes = new(bytes.Buffer)
	s.Logger = log.New(s.LogBytes, "", 0)

	s.SwarmListener = newSwarmListener(
		s.SSListenerMock,
		s.SSClientMock,
		s.SSCacheMock,
		s.SSPoller,
		make(chan Event),
		make(chan Notification),
		s.NodeListeningMock,
		s.NodeClientMock,
		s.NodeCacheMock,
		make(chan Event),
		make(chan Notification),
		s.NotifyDistributorMock,
		NewCancelManager(true),
		NewCancelManager(true),
		NewCancelManager(true),
		NewCancelManager(true),
		false,
		true,
		"com.df.notify",
		"com.docker.stack.namespace",
		s.Logger,
	)
}

func (s *SwarmListenerTestSuite) Test_Run_ServicesChannel() {
	s.SwarmListener.IncludeNodeInfo = true

	s1NodeInfo := NodeIPSet{}
	s1NodeInfo.Add("node1", "10.0.0.1", "node1id")

	receivedBothNotifications := make(chan struct{})
	ss1 := SwarmService{swarm.Service{ID: "serviceID1",
		Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "serviceName1"}}}, s1NodeInfo}

	ss1m := SwarmServiceMini{ID: "serviceID1", Name: "serviceName1", Labels: map[string]string{}, NodeInfo: s1NodeInfo}
	ss2m := SwarmServiceMini{ID: "serviceID2", Name: "serviceName2", Labels: map[string]string{}}

	s.SSListenerMock.On("ListenForServiceEvents", mock.AnythingOfType("chan<- service.Event"))
	s.SSClientMock.On("SwarmServiceInspect", mock.AnythingOfType("*context.cancelCtx"), "serviceID1", true).Return(&ss1, nil)
	s.SSCacheMock.On("InsertAndCheck", ss1m).Return(true).
		On("Get", "serviceID2").Return(ss2m, true).
		On("Len").Return(2)
	s.NotifyDistributorMock.
		On("HasServiceListeners").Return(true).
		On("HasNodeListeners").Return(false).
		On("Run", mock.AnythingOfType("<-chan service.Notification"), mock.AnythingOfType("<-chan service.Notification"))
	s.SSPoller.
		On("Run", mock.AnythingOfType("chan<- service.Event"))

	s.SwarmListener.Run()

	go func() {
		s.SwarmListener.SSEventChan <- Event{
			ID:       "serviceID1",
			Type:     EventTypeCreate,
			TimeNano: int64(1),
			UseCache: true,
		}
	}()

	go func() {
		s.SwarmListener.SSEventChan <- Event{
			ID:       "serviceID2",
			Type:     EventTypeRemove,
			TimeNano: int64(2),
			UseCache: true,
		}
	}()

	notificationMap := map[string]Notification{}

	go func() {
		for {
			select {
			case n := <-s.SwarmListener.SSNotificationChan:
				notificationMap[n.ID] = n
				if len(notificationMap) == 2 {
					receivedBothNotifications <- struct{}{}
					return
				}
			}
		}
	}()

	timeout := time.NewTimer(time.Second * 5).C

	for {
		if receivedBothNotifications == nil {
			break
		}
		select {
		case <-receivedBothNotifications:
			receivedBothNotifications = nil
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Len(notificationMap, 2)
	s.Equal(int64(1), notificationMap["serviceID1"].TimeNano)
	s.Equal(int64(2), notificationMap["serviceID2"].TimeNano)
	s.Equal(EventTypeCreate, notificationMap["serviceID1"].EventType)
	s.Equal(EventTypeRemove, notificationMap["serviceID2"].EventType)
	s.SSListenerMock.AssertExpectations(s.T())
	s.SSClientMock.AssertExpectations(s.T())
	s.SSCacheMock.AssertExpectations(s.T())
	s.NotifyDistributorMock.AssertExpectations(s.T())
	s.SSPoller.AssertExpectations(s.T())

}

func (s *SwarmListenerTestSuite) Test_Run_NodeChannel() {

	n1 := swarm.Node{ID: "nodeID1",
		Description: swarm.NodeDescription{
			Hostname: "node1Hostname",
		}}
	n1m := NodeMini{ID: "nodeID1",
		EngineLabels: map[string]string{},
		NodeLabels:   map[string]string{},
		Hostname:     "node1Hostname",
	}
	n2m := NodeMini{ID: "nodeID2",
		EngineLabels: map[string]string{},
		NodeLabels:   map[string]string{},
		Hostname:     "node2Hostname",
	}

	s.NodeListeningMock.On("ListenForNodeEvents", mock.AnythingOfType("chan<- service.Event"))
	s.NodeClientMock.On("NodeInspect", "nodeID1").Return(n1, nil)
	s.NodeCacheMock.On("InsertAndCheck", n1m).Return(true).
		On("Get", "nodeID2").Return(n2m, true)
	s.NotifyDistributorMock.
		On("HasServiceListeners").Return(false).
		On("HasNodeListeners").Return(true).
		On("Run", mock.AnythingOfType("<-chan service.Notification"), mock.AnythingOfType("<-chan service.Notification"))

	s.SwarmListener.Run()

	go func() {
		s.SwarmListener.NodeEventChan <- Event{
			ID:       "nodeID1",
			Type:     EventTypeCreate,
			TimeNano: int64(1),
			UseCache: true,
		}
	}()

	go func() {
		s.SwarmListener.NodeEventChan <- Event{
			ID:       "nodeID2",
			Type:     EventTypeRemove,
			TimeNano: int64(2),
			UseCache: true,
		}
	}()

	notificationMap := map[string]Notification{}
	timeout := time.NewTimer(time.Second * 5).C

L:
	for {
		select {
		case n := <-s.SwarmListener.NodeNotificationChan:
			notificationMap[n.ID] = n
			if len(notificationMap) == 2 {
				break L
			}
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Len(notificationMap, 2)
	s.Equal(int64(1), notificationMap["nodeID1"].TimeNano)
	s.Equal(int64(2), notificationMap["nodeID2"].TimeNano)
	s.Equal(EventTypeCreate, notificationMap["nodeID1"].EventType)
	s.Equal(EventTypeRemove, notificationMap["nodeID2"].EventType)
	s.NodeListeningMock.AssertExpectations(s.T())
	s.NodeClientMock.AssertExpectations(s.T())
	s.NodeCacheMock.AssertExpectations(s.T())
	s.NotifyDistributorMock.AssertExpectations(s.T())

}

func (s *SwarmListenerTestSuite) Test_NotifyServices_WithCache() {

	expServices := []SwarmService{
		{
			swarm.Service{
				ID: "serviceID1"}, nil,
		},
		{
			swarm.Service{
				ID: "serviceID2"}, nil,
		},
	}
	s.SSClientMock.On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx")).Return(expServices, nil)
	s.NotifyDistributorMock.On("HasServiceListeners").Return(true)

	s.SwarmListener.NotifyServices(true)

	timeout := time.NewTimer(time.Second * 5).C

	eventCnt := 0

	for {
		if eventCnt == 2 {
			break
		}
		select {
		case e := <-s.SwarmListener.SSEventChan:
			s.True(e.UseCache)
			eventCnt++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, eventCnt)
	s.SSClientMock.AssertExpectations(s.T())
}

func (s *SwarmListenerTestSuite) Test_NotifyServices_WithoutCache() {

	expServices := []SwarmService{
		{
			swarm.Service{
				ID: "serviceID1"}, nil,
		},
		{
			swarm.Service{
				ID: "serviceID2"}, nil,
		},
	}
	s.SSClientMock.On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx")).Return(expServices, nil)
	s.NotifyDistributorMock.On("HasServiceListeners").Return(true)

	s.SwarmListener.NotifyServices(false)

	timeout := time.NewTimer(time.Second * 5).C

	eventCnt := 0

	for {
		if eventCnt == 2 {
			break
		}
		select {
		case e := <-s.SwarmListener.SSEventChan:
			s.False(e.UseCache)
			eventCnt++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, eventCnt)
	s.SSClientMock.AssertExpectations(s.T())
}

func (s *SwarmListenerTestSuite) Test_NotifyNodes_WithoutCache() {
	expNodes := []swarm.Node{
		{
			ID: "nodeID1",
		},
		{
			ID: "nodeID2",
		},
	}
	s.NodeClientMock.On("NodeList", mock.AnythingOfType("*context.emptyCtx")).Return(expNodes, nil)
	s.NotifyDistributorMock.On("HasNodeListeners").Return(true)

	s.SwarmListener.NotifyNodes(false)

	timeout := time.NewTimer(time.Second * 5).C

	eventCnt := 0

	for {
		if eventCnt == 2 {
			break
		}
		select {
		case e := <-s.SwarmListener.NodeEventChan:
			s.False(e.UseCache)
			eventCnt++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, eventCnt)
	s.NodeClientMock.AssertExpectations(s.T())
}

func (s *SwarmListenerTestSuite) Test_NotifyNodes_WithCache() {
	expNodes := []swarm.Node{
		{
			ID: "nodeID1",
		},
		{
			ID: "nodeID2",
		},
	}
	s.NodeClientMock.On("NodeList", mock.AnythingOfType("*context.emptyCtx")).Return(expNodes, nil)
	s.NotifyDistributorMock.On("HasNodeListeners").Return(true)

	s.SwarmListener.NotifyNodes(true)

	timeout := time.NewTimer(time.Second * 5).C

	eventCnt := 0

	for {
		if eventCnt == 2 {
			break
		}
		select {
		case e := <-s.SwarmListener.NodeEventChan:
			s.True(e.UseCache)
			eventCnt++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, eventCnt)
	s.NodeClientMock.AssertExpectations(s.T())
}

func (s *SwarmListenerTestSuite) Test_GetServices_WithoutNodeInfo() {

	expServices := []SwarmService{
		{
			swarm.Service{ID: "serviceID1"}, nil,
		},
		{
			swarm.Service{ID: "serviceID2"}, nil,
		},
	}
	s.SSClientMock.On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx")).Return(expServices, nil)

	params, err := s.SwarmListener.GetServicesParameters(context.Background())
	s.Require().NoError(err)
	s.Len(params, 2)

	s.SSClientMock.AssertExpectations(s.T())
}

func (s *SwarmListenerTestSuite) Test_GetServices_WithNodeInfo() {

	s.SwarmListener.IncludeNodeInfo = true

	s1NodeInfo := NodeIPSet{}
	s1NodeInfo.Add("node1", "10.0.0.1", "node1id")

	s2NodeInfo := NodeIPSet{}
	s2NodeInfo.Add("node2", "10.0.1.1", "node2id")

	s1 := SwarmService{swarm.Service{ID: "serviceID1"}, nil}
	s2 := SwarmService{swarm.Service{ID: "serviceID2"}, nil}

	expServices := []SwarmService{s1, s2}
	s.SSClientMock.On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx")).
		Return(expServices, nil).
		On("GetNodeInfo", mock.AnythingOfType("*context.emptyCtx"), s1).
		Return(s1NodeInfo, nil).
		On("GetNodeInfo", mock.AnythingOfType("*context.emptyCtx"), s2).
		Return(s2NodeInfo, nil)

	params, err := s.SwarmListener.GetServicesParameters(context.Background())
	s.Require().NoError(err)
	s.Len(params, 2)

	s.SSClientMock.AssertExpectations(s.T())
}

func (s *SwarmListenerTestSuite) Test_GetNodes() {

	expServices := []swarm.Node{
		{ID: "nodeID1"},
		{ID: "nodeID2"},
	}
	s.NodeClientMock.On("NodeList", mock.AnythingOfType("*context.emptyCtx")).Return(expServices, nil)

	params, err := s.SwarmListener.GetNodesParameters(context.Background())
	s.Require().NoError(err)
	s.Len(params, 2)

	s.NodeClientMock.AssertExpectations(s.T())
}
