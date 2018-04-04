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

	NotifyDistributorMock *notifyDistributorMock

	ServiceCancelManagerMock *cancelManagingMock
	SwarmListener            *SwarmListener
	Logger                   *log.Logger
	LogBytes                 *bytes.Buffer
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
	s.NotifyDistributorMock = new(notifyDistributorMock)
	s.ServiceCancelManagerMock = new(cancelManagingMock)
	s.LogBytes = new(bytes.Buffer)
	s.Logger = log.New(s.LogBytes, "", 0)

	s.SwarmListener = newSwarmListener(
		s.SSListenerMock,
		s.SSClientMock,
		s.SSCacheMock,
		s.NodeListeningMock,
		s.NodeClientMock,
		s.NodeCacheMock,
		s.NotifyDistributorMock,
		s.ServiceCancelManagerMock,
		true,
		"com.df.notify",
		"com.docker.stack.namespace",
		s.Logger,
	)
}

func (s *SwarmListenerTestSuite) Test_Run_ServicesChannel() {
	s.SwarmListener.IncludeNodeInfo = false

	notifyCnt := 0
	done := make(chan struct{})
	ss1 := SwarmService{swarm.Service{ID: "serviceID1",
		Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "serviceName1"}}}, nil}

	ss1m := SwarmServiceMini{ID: "serviceID1", Name: "serviceName1", Labels: map[string]string{}}
	ss2m := SwarmServiceMini{ID: "serviceID2", Name: "serviceName2", Labels: map[string]string{}}

	s.SSListenerMock.On("ListenForServiceEvents", mock.AnythingOfType("chan<- service.Event"))
	s.SSClientMock.On("SwarmServiceInspect", mock.AnythingOfType("*context.emptyCtx"), "serviceID1", false).Return(&ss1, nil)
	s.SSCacheMock.On("InsertAndCheck", ss1m).Return(true).
		On("Get", "serviceID2").Return(ss2m, true).
		On("Delete", "serviceID2").
		On("Len").Return(2)
	s.NotifyDistributorMock.
		On("HasServiceListeners").Return(true).
		On("HasNodeListeners").Return(false).
		On("Run", mock.AnythingOfType("<-chan service.Notification"), mock.AnythingOfType("<-chan service.Notification")).Run(func(args mock.Arguments) {
		nChan := args.Get(0).(<-chan Notification)
		go func() {
			for range nChan {
				notifyCnt++
				if notifyCnt == 2 {
					done <- struct{}{}
				}
			}
		}()
	})
	s.ServiceCancelManagerMock.
		On("Add", "serviceID1", mock.AnythingOfType("int64")).Return(context.Background()).
		On("Delete", "serviceID1", mock.AnythingOfType("int64")).Return(true).
		On("ForceDelete", "serviceID2").Return(true)

	s.SwarmListener.Run()

	go func() {
		s.SwarmListener.SSEventChan <- Event{
			ID:   "serviceID1",
			Type: EventTypeCreate,
		}
	}()

	go func() {
		s.SwarmListener.SSEventChan <- Event{
			ID:   "serviceID2",
			Type: EventTypeRemove,
		}
	}()

	timeout := time.NewTimer(time.Second * 5).C

	for {
		if done == nil {
			break
		}
		select {
		case <-done:
			done = nil
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, notifyCnt)
	s.SSListenerMock.AssertExpectations(s.T())
	s.SSClientMock.AssertExpectations(s.T())
	s.SSCacheMock.AssertExpectations(s.T())
	s.NotifyDistributorMock.AssertExpectations(s.T())
	s.ServiceCancelManagerMock.AssertExpectations(s.T())

}

func (s *SwarmListenerTestSuite) Test_Run_NodeChannel() {

	notifyCnt := 0
	done := make(chan struct{})
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
		On("Get", "nodeID2").Return(n2m, true).
		On("Delete", "nodeID2")
	s.NotifyDistributorMock.
		On("HasServiceListeners").Return(false).
		On("HasNodeListeners").Return(true).
		On("Run", mock.AnythingOfType("<-chan service.Notification"), mock.AnythingOfType("<-chan service.Notification")).
		Run(func(args mock.Arguments) {
			nChan := args.Get(1).(<-chan Notification)
			go func() {
				for range nChan {
					notifyCnt++
					if notifyCnt == 2 {
						done <- struct{}{}
					}
				}
			}()
		})

	s.SwarmListener.Run()

	go func() {
		s.SwarmListener.NodeEventChan <- Event{
			ID:   "nodeID1",
			Type: EventTypeCreate,
		}
	}()

	go func() {
		s.SwarmListener.NodeEventChan <- Event{
			ID:   "nodeID2",
			Type: EventTypeRemove,
		}
	}()

	timeout := time.NewTimer(time.Second * 5).C

	for {
		if done == nil {
			break
		}
		select {
		case <-done:
			done = nil
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, notifyCnt)
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
	s.SSClientMock.On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx"), true).Return(expServices, nil)

	s.SwarmListener.NotifyServices(true)

	timeout := time.NewTimer(time.Second * 5).C

	eventCnt := 0

	for {
		if eventCnt == 2 {
			break
		}
		select {
		case <-s.SwarmListener.SSEventChan:
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
			swarm.Service{ID: "serviceID1"}, nil,
		},
		{
			swarm.Service{ID: "serviceID2"}, nil,
		},
	}
	s.SSClientMock.On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx"), true).Return(expServices, nil)

	s.SwarmListener.NotifyServices(false)

	timeout := time.NewTimer(time.Second * 5).C

	notificationCnt := 0

	for {
		if notificationCnt == 2 {
			break
		}
		select {
		case <-s.SwarmListener.SSNotificationChan:
			notificationCnt++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, notificationCnt)
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

	s.SwarmListener.NotifyNodes(false)

	timeout := time.NewTimer(time.Second * 5).C

	notificationCnt := 0

	for {
		if notificationCnt == 2 {
			break
		}
		select {
		case <-s.SwarmListener.NodeNotificationChan:
			notificationCnt++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, notificationCnt)
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

	s.SwarmListener.NotifyNodes(true)

	timeout := time.NewTimer(time.Second * 5).C

	eventCnt := 0

	for {
		if eventCnt == 2 {
			break
		}
		select {
		case <-s.SwarmListener.NodeEventChan:
			eventCnt++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, eventCnt)
	s.NodeClientMock.AssertExpectations(s.T())
}

func (s *SwarmListenerTestSuite) Test_GetServices() {

	expServices := []SwarmService{
		{
			swarm.Service{ID: "serviceID1"}, nil,
		},
		{
			swarm.Service{ID: "serviceID2"}, nil,
		},
	}
	s.SSClientMock.On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx"), true).Return(expServices, nil)

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
