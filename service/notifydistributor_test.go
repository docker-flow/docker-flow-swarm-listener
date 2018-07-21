package service

import (
	"bytes"
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type NotifyDistributorTestSuite struct {
	suite.Suite
	log      *log.Logger
	logBytes *bytes.Buffer
	ctx      context.Context
}

func TestNotifyDistributorUnitTestSuite(t *testing.T) {
	suite.Run(t, new(NotifyDistributorTestSuite))
}

func (s *NotifyDistributorTestSuite) SetupSuite() {
	s.logBytes = new(bytes.Buffer)
	s.log = log.New(s.logBytes, "", 0)
	s.ctx = context.Background()
}
func (s *NotifyDistributorTestSuite) SetupTest() {
	s.logBytes.Reset()
}

func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromStrings() {
	notifyD := newNotifyDistributorfromStrings(
		"http://host1:8080/recofigureservice,http://host2:8080/recofigureservice",
		"http://host1:8080/removeservice,http://host2:8080/removeservice",
		"http://host1:8080/reconfigurenode",
		"http://host2:8080/removenode",
		5, 10, s.log)

	s.Len(notifyD.NotifyEndpoints, 2)
	host1EP, ok := notifyD.NotifyEndpoints["host1:8080"]

	s.Require().True(ok)

	s.AssertEndpoints(
		host1EP,
		"http://host1:8080/recofigureservice",
		"http://host1:8080/removeservice",
		"http://host1:8080/reconfigurenode",
		"",
	)

	host2EP, ok := notifyD.NotifyEndpoints["host2:8080"]
	s.Require().True(ok)
	s.AssertEndpoints(
		host2EP,
		"http://host2:8080/recofigureservice",
		"http://host2:8080/removeservice",
		"",
		"http://host2:8080/removenode",
	)

	s.True(notifyD.HasServiceListeners())
	s.True(notifyD.HasNodeListeners())
}

func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromStringsWithParameters() {
	notifyD := newNotifyDistributorfromStrings(
		"http://host1:8080/recofigureservice?hello=world,http://host2:8080/recofigureservice",
		"http://host1:8080/removeservice,http://host2:8080/removeservice",
		"http://host1:8080/reconfigurenode?dog=cat&bear=fox",
		"http://host2:8080/removenode?service=aws",
		5, 10, s.log)

	s.Len(notifyD.NotifyEndpoints, 2)
	host1EP, ok := notifyD.NotifyEndpoints["host1:8080"]

	s.Require().True(ok)

	s.AssertEndpoints(
		host1EP,
		"http://host1:8080/recofigureservice?hello=world",
		"http://host1:8080/removeservice",
		"http://host1:8080/reconfigurenode?dog=cat&bear=fox",
		"",
	)

	host2EP, ok := notifyD.NotifyEndpoints["host2:8080"]
	s.Require().True(ok)
	s.AssertEndpoints(
		host2EP,
		"http://host2:8080/recofigureservice",
		"http://host2:8080/removeservice",
		"",
		"http://host2:8080/removenode?service=aws",
	)

	s.True(notifyD.HasServiceListeners())
	s.True(notifyD.HasNodeListeners())
}
func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromStrings_SeparateListeners() {
	notifyD := newNotifyDistributorfromStrings(
		"http://host1:8080/recofigure1",
		"http://host1:8080/removeservice",
		"http://host2:8080/reconfigurenode",
		"http://host2/removenode1,http://host2:8080/removenode2",
		5, 10, s.log)

	s.Len(notifyD.NotifyEndpoints, 3)
	host1EP, ok := notifyD.NotifyEndpoints["host1:8080"]
	s.Require().True(ok)

	s.AssertEndpoints(
		host1EP,
		"http://host1:8080/recofigure1",
		"http://host1:8080/removeservice",
		"",
		"",
	)

	host28080EP, ok := notifyD.NotifyEndpoints["host2:8080"]
	s.Require().True(ok)
	s.AssertEndpoints(
		host28080EP,
		"",
		"",
		"http://host2:8080/reconfigurenode",
		"http://host2:8080/removenode2",
	)

	host2EP, ok := notifyD.NotifyEndpoints["host2"]
	s.Require().True(ok)
	s.AssertEndpoints(
		host2EP,
		"",
		"",
		"",
		"http://host2/removenode1",
	)

	s.True(notifyD.HasServiceListeners())
	s.True(notifyD.HasNodeListeners())
}

func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromStrings_JustSwarmListeners() {
	notifyD := newNotifyDistributorfromStrings(
		"http://host1:8080/recofigure1",
		"http://host1:8080/removeservice", "", "",
		5, 10, s.log)

	s.Len(notifyD.NotifyEndpoints, 1)
	host1EP, ok := notifyD.NotifyEndpoints["host1:8080"]
	s.Require().True(ok)

	s.AssertEndpoints(
		host1EP,
		"http://host1:8080/recofigure1",
		"http://host1:8080/removeservice",
		"",
		"",
	)

	s.True(notifyD.HasServiceListeners())
	s.False(notifyD.HasNodeListeners())
}
func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromStrings_JustNodeListeners() {
	notifyD := newNotifyDistributorfromStrings(
		"", "",
		"http://host2:8080/reconfigurenode",
		"http://host2:8080/removenode1,http://host2/removenode2",
		5, 10, s.log)

	s.Len(notifyD.NotifyEndpoints, 2)
	host28080EP, ok := notifyD.NotifyEndpoints["host2:8080"]
	s.Require().True(ok)
	s.AssertEndpoints(
		host28080EP,
		"",
		"",
		"http://host2:8080/reconfigurenode",
		"http://host2:8080/removenode1",
	)

	host2EP, ok := notifyD.NotifyEndpoints["host2"]
	s.Require().True(ok)
	s.AssertEndpoints(
		host2EP,
		"",
		"",
		"",
		"http://host2/removenode2",
	)

	s.False(notifyD.HasServiceListeners())
	s.True(notifyD.HasNodeListeners())
}

func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromEnv_ServiceCreate() {
	envKeys := []string{"DF_NOTIFY_CREATE_SERVICE_URL",
		"DF_NOTIF_CREATE_SERVICE_URL",
		"DF_NOTIFICATION_URL"}
	for _, envKey := range envKeys {
		oldHost := os.Getenv(envKey)
		os.Setenv(envKey, "http://host1,http://host2")

		notifyD := NewNotifyDistributorFromEnv(5, 10, s.log)

		if notifyD == nil {
			s.Fail("%s returns nil", envKey)
			os.Setenv(envKey, oldHost)
			continue
		}

		s.Len(notifyD.NotifyEndpoints, 2)

		ep1, ok1 := notifyD.NotifyEndpoints["host1"]
		s.True(ok1)

		if ep1.ServiceNotifier == nil {
			s.Fail("%s nil host1 ServiceNotifier", envKey)
			os.Setenv(envKey, oldHost)
			continue
		}

		s.Equal("http://host1", ep1.ServiceNotifier.GetCreateAddr())

		ep2, ok2 := notifyD.NotifyEndpoints["host2"]
		s.True(ok2)

		if ep2.ServiceNotifier == nil {
			s.Fail("%s is nil host2 ServiceNotifier", envKey)
			os.Setenv(envKey, oldHost)
			continue
		}
		s.Equal("http://host2", ep2.ServiceNotifier.GetCreateAddr())
		os.Setenv(envKey, oldHost)
	}
}

func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromEnv_ServiceRemove() {
	envKeys := []string{"DF_NOTIFY_REMOVE_SERVICE_URL",
		"DF_NOTIF_REMOVE_SERVICE_URL",
		"DF_NOTIFICATION_URL"}
	for _, envKey := range envKeys {
		oldHost := os.Getenv(envKey)
		os.Setenv(envKey, "http://host1,http://host2")

		notifyD := NewNotifyDistributorFromEnv(5, 10, s.log)

		if notifyD == nil {
			s.Fail("%s returns nil", envKey)
			os.Setenv(envKey, oldHost)
			continue
		}

		s.Len(notifyD.NotifyEndpoints, 2)

		ep1, ok1 := notifyD.NotifyEndpoints["host1"]
		s.True(ok1)

		if ep1.ServiceNotifier == nil {
			s.Fail("%s nil host1 ServiceNotifier", envKey)
			os.Setenv(envKey, oldHost)
			continue
		}

		s.Equal("http://host1", ep1.ServiceNotifier.GetRemoveAddr())

		ep2, ok2 := notifyD.NotifyEndpoints["host2"]
		s.True(ok2)

		if ep2.ServiceNotifier == nil {
			s.Fail("%s nil host2 ServiceNotifier", envKey)
			os.Setenv(envKey, oldHost)
			continue
		}
		s.Equal("http://host2", ep2.ServiceNotifier.GetRemoveAddr())
		os.Setenv(envKey, oldHost)
	}
}

func (s *NotifyDistributorTestSuite) Test_NewNotifyDistributorFromEnv_Node() {
	defer func() {
		os.Unsetenv("DF_NOTIFY_CREATE_NODE_URL")
		os.Unsetenv("DF_NOTIFY_REMOVE_NODE_URL")
	}()
	os.Setenv("DF_NOTIFY_CREATE_NODE_URL", "http://host1/create,http://host2/create")
	os.Setenv("DF_NOTIFY_REMOVE_NODE_URL", "http://host1/remove,http://host2/remove")

	notifyD := NewNotifyDistributorFromEnv(5, 10, s.log)
	s.Require().NotNil(notifyD)

	s.Len(notifyD.NotifyEndpoints, 2)
	ep1, ok1 := notifyD.NotifyEndpoints["host1"]
	s.True(ok1)

	s.Require().NotNil(ep1.NodeNotifier)
	s.Equal("http://host1/create", ep1.NodeNotifier.GetCreateAddr())
	s.Equal("http://host1/remove", ep1.NodeNotifier.GetRemoveAddr())

	ep2, ok2 := notifyD.NotifyEndpoints["host2"]
	s.True(ok2)

	s.Require().NotNil(ep2.NodeNotifier)
	s.Equal("http://host2/create", ep2.NodeNotifier.GetCreateAddr())
	s.Equal("http://host2/remove", ep2.NodeNotifier.GetRemoveAddr())
}

func (s *NotifyDistributorTestSuite) Test_RunDistributesNotificationsToEndpoints_Servies() {

	service1ErrChan := make(chan error)
	service2ErrChan := make(chan error)

	serviceNotifyMock1 := notificationSenderMock{}
	serviceNotifyMock1.On("Create", mock.AnythingOfType("*context.cancelCtx"), "hello=world").
		Return(nil).Run(func(args mock.Arguments) {

	})
	serviceNotifyMock1.On("Remove", mock.AnythingOfType("*context.cancelCtx"), "hello=world2").
		Return(nil)
	serviceNotifyMock2 := notificationSenderMock{}
	serviceNotifyMock2.On("Create", mock.AnythingOfType("*context.cancelCtx"), "hello=world").
		Return(nil)
	serviceNotifyMock2.On("Remove", mock.AnythingOfType("*context.cancelCtx"), "hello=world2").
		Return(nil)

	endpoints := map[string]NotifyEndpoint{
		"host1": {
			ServiceNotifier: &serviceNotifyMock1,
			NodeNotifier:    nil,
		},
		"host2": {
			ServiceNotifier: &serviceNotifyMock2,
			NodeNotifier:    nil,
		},
	}

	notifyD := newNotifyDistributor(endpoints, NewCancelManager(),
		NewCancelManager(), 1, s.log)
	serviceChan := make(chan Notification)

	notifyD.Run(serviceChan, nil)

	go func() {
		serviceChan <- Notification{
			EventType:  EventTypeCreate,
			ID:         "sid1",
			Parameters: "hello=world",
			TimeNano:   int64(1),
			Context:    s.ctx,
			ErrorChan:  service1ErrChan,
		}
	}()
	go func() {
		serviceChan <- Notification{
			EventType:  EventTypeRemove,
			ID:         "sid2",
			Parameters: "hello=world2",
			TimeNano:   int64(2),
			Context:    s.ctx,
			ErrorChan:  service2ErrChan,
		}
	}()

	timer := time.NewTimer(time.Second * 5).C

	for {
		if service1ErrChan == nil && service2ErrChan == nil {
			break
		}
		select {
		case <-service1ErrChan:
			service1ErrChan = nil
		case <-service2ErrChan:
			service2ErrChan = nil
		case <-timer:
			s.Fail("Timeout")
			return
		}
	}

	serviceNotifyMock1.AssertExpectations(s.T())
	serviceNotifyMock2.AssertExpectations(s.T())
}

func (s *NotifyDistributorTestSuite) Test_RunDistributesNotificationsToEndpoints_Nodes1() {
	node1ErrChan := make(chan error)
	node2ErrChan := make(chan error)

	nodesNotifyMock1 := notificationSenderMock{}
	nodesNotifyMock1.On("Create", mock.AnythingOfType("*context.cancelCtx"), "hello=world").
		Return(nil)
	nodesNotifyMock1.On("Remove", mock.AnythingOfType("*context.cancelCtx"), "hello=world2").
		Return(nil)

	nodesNotifyMock2 := notificationSenderMock{}
	nodesNotifyMock2.On("Create", mock.AnythingOfType("*context.cancelCtx"), "hello=world").
		Return(nil)
	nodesNotifyMock2.On("Remove", mock.AnythingOfType("*context.cancelCtx"), "hello=world2").
		Return(nil)

	endpoints := map[string]NotifyEndpoint{
		"host1": {
			ServiceNotifier: nil,
			NodeNotifier:    &nodesNotifyMock1,
		},
		"host2": {
			ServiceNotifier: nil,
			NodeNotifier:    &nodesNotifyMock2,
		},
	}

	notifyD := newNotifyDistributor(endpoints, NewCancelManager(),
		NewCancelManager(), 1, s.log)
	nodeChan := make(chan Notification)

	notifyD.Run(nil, nodeChan)

	go func() {
		nodeChan <- Notification{
			EventType:  EventTypeCreate,
			ID:         "nid1",
			Parameters: "hello=world",
			TimeNano:   int64(1),
			Context:    s.ctx,
			ErrorChan:  node1ErrChan,
		}
	}()
	go func() {
		nodeChan <- Notification{
			EventType:  EventTypeRemove,
			ID:         "nid2",
			Parameters: "hello=world2",
			TimeNano:   int64(2),
			Context:    s.ctx,
			ErrorChan:  node2ErrChan,
		}
	}()

	timer := time.NewTimer(time.Second * 5).C

	for {
		if node1ErrChan == nil && node2ErrChan == nil {
			break
		}
		select {
		case <-node1ErrChan:
			node1ErrChan = nil
		case <-node2ErrChan:
			node2ErrChan = nil
		case <-timer:
			s.Fail("Timeout")
			return
		}
	}

	nodesNotifyMock1.AssertExpectations(s.T())
	nodesNotifyMock2.AssertExpectations(s.T())
}

func (s *NotifyDistributorTestSuite) AssertEndpoints(endpoint NotifyEndpoint, serviceCreateAddr, serviceRemoveAddr, nodeCreateAddr, nodeRemoveAddr string) {
	if len(serviceCreateAddr) == 0 && len(serviceRemoveAddr) == 0 {
		s.Nil(endpoint.ServiceNotifier)
	} else {
		s.Require().NotNil(endpoint.ServiceNotifier)
		s.Equal(serviceCreateAddr, endpoint.ServiceNotifier.GetCreateAddr())
		s.Equal(serviceRemoveAddr, endpoint.ServiceNotifier.GetRemoveAddr())
	}
	if len(nodeCreateAddr) == 0 && len(nodeRemoveAddr) == 0 {
		s.Nil(endpoint.NodeNotifier)
	} else {
		s.Require().NotNil(endpoint.NodeNotifier)
		s.Equal(nodeCreateAddr, endpoint.NodeNotifier.GetCreateAddr())
		s.Equal(nodeRemoveAddr, endpoint.NodeNotifier.GetRemoveAddr())
	}

}
