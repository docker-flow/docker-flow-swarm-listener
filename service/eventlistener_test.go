package service

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type EventListenerTestSuite struct {
	suite.Suite
	serviceName string
}

func TestEventListenerUnitTestSuite(t *testing.T) {
	s := new(EventListenerTestSuite)
	s.serviceName = "my-service"
	logPrintfOrig := logPrintf
	defer func() {
		logPrintf = logPrintfOrig
		os.Unsetenv("DF_NOTIFY_LABEL")
	}()
	logPrintf = func(format string, v ...interface{}) {}
	os.Setenv("DF_NOTIFY_LABEL", "com.df.notify")

	suite.Run(t, s)
}

func (s *EventListenerTestSuite) Test_ListenForEvents_IncorrectSocket() {
	eventListener := NewEventListener("unix:///this/socket/does/not/exist")
	_, errs := eventListener.ListenForEvents()

	err := s.GetChannelError(errs)
	s.Error(err)
}

func (s *EventListenerTestSuite) Test_ListenForEvents_CreateService() {
	eventListener := NewEventListener("unix:///var/run/docker.sock")
	service := NewService("unix:///var/run/docker.sock")

	events, errs := eventListener.ListenForEvents()

	defer func() {
		removeTestService("util-el-1")
	}()
	createTestService("util-el-1", []string{"com.df.notify=true", "com.df.servicePath=/demo", "com.df.distribute=true"}, "", "")

	event, err := s.GetChannelEvent(events, errs)

	s.Require().NoError(err)

	s.Equal("create", event.Action)

	serviceID := event.ServiceID
	s.Require().NotEmpty(serviceID)

	eventServices, err := service.GetServicesFromID(serviceID)
	s.Require().NoError(err)
	s.Require().NotNil(eventServices)
	s.Require().Len(*eventServices, 1)

	eventService := (*eventServices)[0]

	s.Equal("util-el-1", eventService.Spec.Name)
	s.Nil(eventService.NodeInfo)
}

func (s *EventListenerTestSuite) Test_ListenForEvents_CreateService_WithNodeInfo() {

	eventListener := NewEventListener("unix:///var/run/docker.sock")
	service := NewService("unix:///var/run/docker.sock")

	events, errs := eventListener.ListenForEvents()

	defer func() {
		os.Unsetenv("DF_INCLUDE_NODE_IP_INFO")
		removeTestService("util-el-1")
		removeTestNetwork("util-el-network")
	}()
	os.Setenv("DF_INCLUDE_NODE_IP_INFO", "true")
	createTestNetwork("util-el-network")
	createTestService("util-el-1",
		[]string{"com.df.notify=true", "com.df.scrapeNetwork=util-el-network", "com.df.distribute=true"},
		"", "util-el-network")

	event, err := s.GetChannelEvent(events, errs)

	s.Require().NoError(err)

	s.Equal("create", event.Action)

	serviceID := event.ServiceID
	s.Require().NotEmpty(serviceID)

	eventServices, err := service.GetServicesFromID(serviceID)
	s.Require().NoError(err)
	s.Require().NotNil(eventServices)
	s.Require().Len(*eventServices, 1)

	eventService := (*eventServices)[0]

	s.Equal("util-el-1", eventService.Spec.Name)
	s.NotNil(eventService.NodeInfo)
}
func (s *EventListenerTestSuite) Test_ListenForEvents_RemoveService() {
	eventListener := NewEventListener("unix:///var/run/docker.sock")
	service := NewService("unix:///var/run/docker.sock")

	events, errs := eventListener.ListenForEvents()

	defer func() {
		removeTestService("util-el-1")
	}()
	createTestService("util-el-1", []string{"com.df.notify=true", "com.df.servicePath=/demo", "com.df.distribute=true"}, "", "")

	// Check create event action
	event, err := s.GetChannelEvent(events, errs)
	s.Require().NoError(err)
	s.Equal("create", event.Action)

	serviceID := event.ServiceID
	s.Require().NotEmpty(serviceID)

	eventServices, err := service.GetServicesFromID(serviceID)
	s.Require().NoError(err)
	s.Require().NotNil(eventServices)
	s.Require().Len(*eventServices, 1)
	eventService := (*eventServices)[0]

	removeTestService("util-el-1")
	event, err = s.GetChannelEvent(events, errs)

	s.Require().NoError(err)
	s.Equal("remove", event.Action)

	serviceID = event.ServiceID
	s.NotEmpty(serviceID)
	s.Equal(eventService.ID, serviceID)
}

func (s *EventListenerTestSuite) GetChannelError(errs <-chan error) error {
	timeOut := time.NewTimer(time.Second * 5).C
	for {
		select {
		case err := <-errs:
			return err
		case <-timeOut:
			return fmt.Errorf("Timeout")
		}
	}
}

func (s *EventListenerTestSuite) GetChannelEvent(events <-chan Event, errs <-chan error) (*Event, error) {
	timeOut := time.NewTimer(time.Second * 5).C
	for {
		select {
		case event := <-events:
			return &event, nil
		case err := <-errs:
			return nil, err
		case <-timeOut:
			return nil, fmt.Errorf("Timeout")
		}
	}
}
