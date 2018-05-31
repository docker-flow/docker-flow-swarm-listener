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

type ServicePollerTestSuite struct {
	suite.Suite
	SSClientMock *swarmServiceInspector
	SSCacheMock  *swarmServiceCacherMock
	MinifyFunc   func(ss SwarmService) SwarmServiceMini

	SSPoller *SwarmServicePoller
	Logger   *log.Logger
	LogBytes *bytes.Buffer
}

func TestServicePollerUnitTestSuite(t *testing.T) {
	suite.Run(t, new(ServicePollerTestSuite))
}

func (s *ServicePollerTestSuite) SetupTest() {
	s.SSClientMock = new(swarmServiceInspector)
	s.SSCacheMock = new(swarmServiceCacherMock)

	s.MinifyFunc = func(ss SwarmService) SwarmServiceMini {
		return MinifySwarmService(ss, "com.df.notify", "com.df.scrapeNetwork")
	}
	s.LogBytes = new(bytes.Buffer)
	s.Logger = log.New(s.LogBytes, "", 0)

	s.SSPoller = NewSwarmServicePoller(
		s.SSClientMock,
		s.SSCacheMock,
		1,
		false,
		s.MinifyFunc,
		s.Logger,
	)
}

func (s *ServicePollerTestSuite) Test_Run_NoCache() {

	expServices := []SwarmService{
		{swarm.Service{ID: "serviceID1"}, nil},
		{swarm.Service{ID: "serviceID2"}, nil},
	}
	keys := map[string]struct{}{}
	miniSS1 := SwarmServiceMini{
		ID: "serviceID1", Labels: map[string]string{}}
	miniSS2 := SwarmServiceMini{
		ID: "serviceID2", Labels: map[string]string{}}

	eventChan := make(chan Event)

	s.SSClientMock.
		On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx")).Return(expServices, nil)
	s.SSCacheMock.
		On("Keys").Return(keys).
		On("IsNewOrUpdated", miniSS1).Return(true).
		On("IsNewOrUpdated", miniSS2).Return(true)

	go s.SSPoller.Run(eventChan)

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
	s.SSClientMock.AssertExpectations(s.T())
	s.SSCacheMock.AssertExpectations(s.T())
}

func (s *ServicePollerTestSuite) Test_Run_HalfInCache() {

	expServices := []SwarmService{
		{swarm.Service{ID: "serviceID1"}, nil},
		{swarm.Service{ID: "serviceID2"}, nil},
	}
	miniSS1 := SwarmServiceMini{
		ID: "serviceID1", Labels: map[string]string{}}
	miniSS2 := SwarmServiceMini{
		ID: "serviceID2", Labels: map[string]string{}}

	keys := map[string]struct{}{}
	keys["serviceID1"] = struct{}{}

	eventChan := make(chan Event)

	s.SSClientMock.
		On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx")).Return(expServices, nil)
	s.SSCacheMock.
		On("Keys").Return(keys).
		On("IsNewOrUpdated", miniSS1).Return(false).
		On("IsNewOrUpdated", miniSS2).Return(true)

	go s.SSPoller.Run(eventChan)

	timeout := time.NewTimer(time.Second * 5).C
	var eventCreate *Event
	eventsNum := 0

	for {
		if eventsNum == 1 {
			break
		}
		select {
		case event := <-eventChan:
			if event.ID == "serviceID2" {
				eventCreate = &event
			}
			eventsNum++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(1, eventsNum)
	s.Require().NotNil(eventCreate)

	s.Equal("serviceID2", eventCreate.ID)
	s.SSClientMock.AssertExpectations(s.T())
	s.SSCacheMock.AssertExpectations(s.T())
}

func (s *ServicePollerTestSuite) Test_Run_MoreInCache() {

	expServices := []SwarmService{
		{swarm.Service{ID: "serviceID1"}, nil},
		{swarm.Service{ID: "serviceID2"}, nil},
	}
	miniSS1 := SwarmServiceMini{
		ID: "serviceID1", Labels: map[string]string{}}
	miniSS2 := SwarmServiceMini{
		ID: "serviceID2", Labels: map[string]string{}}

	keys := map[string]struct{}{}
	keys["serviceID1"] = struct{}{}
	keys["serviceID2"] = struct{}{}
	keys["serviceID3"] = struct{}{}

	eventChan := make(chan Event)

	s.SSClientMock.
		On("SwarmServiceList", mock.AnythingOfType("*context.emptyCtx")).Return(expServices, nil)
	s.SSCacheMock.
		On("Keys").Return(keys).
		On("IsNewOrUpdated", miniSS1).Return(true).
		On("IsNewOrUpdated", miniSS2).Return(false)

	go s.SSPoller.Run(eventChan)

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
			if event.ID == "serviceID1" {
				eventCreate = &event
			} else if event.ID == "serviceID3" {
				eventRemove = &event
			}
			eventsNum++
		case <-timeout:
			s.Fail("Timeout")
			return
		}
	}

	s.Equal(2, eventsNum)
	s.Require().NotNil(eventCreate)
	s.Require().NotNil(eventRemove)

	s.Equal("serviceID1", eventCreate.ID)
	s.Equal("serviceID3", eventRemove.ID)
	s.SSClientMock.AssertExpectations(s.T())
	s.SSCacheMock.AssertExpectations(s.T())
}
