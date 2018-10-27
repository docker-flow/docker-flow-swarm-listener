package service

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type SwarmServiceClientTestSuite struct {
	suite.Suite
	SClient  *SwarmServiceClient
	Util1ID  string
	Util2ID  string
	Util3ID  string
	Util4ID  string
	Logger   *log.Logger
	LogBytes *bytes.Buffer
}

func TestSwarmServiceClientTestSuite(t *testing.T) {
	suite.Run(t, new(SwarmServiceClientTestSuite))
}

func (s *SwarmServiceClientTestSuite) SetupSuite() {
	createTestOverlayNetwork("util-network")
	createTestService("util-1", []string{"com.df.notify=true", "com.df.scrapeNetwork=util-network"}, false, "", "util-network")
	createTestService("util-2", []string{}, false, "", "util-network")
	createTestService("util-3", []string{"com.df.notify=true"}, true, "", "util-network")
	createTestService("util-4", []string{"com.df.notify=true", "com.df.scrapeNetwork=util-network"}, false, "2", "util-network")

	time.Sleep(time.Second)
	ID1, err := getServiceID("util-1")
	s.Require().NoError(err)
	s.Util1ID = ID1

	ID2, err := getServiceID("util-2")
	s.Require().NoError(err)
	s.Util2ID = ID2

	ID3, err := getServiceID("util-3")
	s.Require().NoError(err)
	s.Util3ID = ID3

	ID4, err := getServiceID("util-4")
	s.Require().NoError(err)
	s.Util4ID = ID4
}

func (s *SwarmServiceClientTestSuite) SetupTest() {
	c, err := NewDockerClientFromEnv()
	s.Require().NoError(err)

	s.LogBytes = new(bytes.Buffer)
	s.Logger = log.New(s.LogBytes, "", 0)

	s.SClient = NewSwarmServiceClient(c, "com.df.notify=true", "com.df.scrapeNetwork", "", true, s.Logger)
}

func (s *SwarmServiceClientTestSuite) TearDownSuite() {
	removeTestService("util-1")
	removeTestService("util-2")
	removeTestService("util-3")
	removeTestService("util-4")
	removeTestNetwork("util-network")
}

func (s *SwarmServiceClientTestSuite) Test_SwarmServiceInspect_NodeInfo_UndefinedScrapeNetwork() {

	util3Service, err := s.SClient.SwarmServiceInspect(context.Background(), s.Util3ID, true)
	s.Require().NoError(err)
	s.Require().NotNil(util3Service)

	s.Equal(s.Util3ID, util3Service.ID)
	s.Nil(util3Service.NodeInfo)
}
func (s *SwarmServiceClientTestSuite) Test_SwarmServiceInspect_With_Service_Name_Prefix() {
	s.SClient.ServiceNamePrefix = "dev1"

	util1Service, err := s.SClient.SwarmServiceInspect(context.Background(), s.Util1ID, false)
	s.Require().NoError(err)
	s.Require().NotNil(util1Service)

	s.Equal(s.Util1ID, util1Service.ID)
	s.Require().Nil(util1Service.NodeInfo)
	s.Equal("dev1_util-1", util1Service.Spec.Name)
}

func (s *SwarmServiceClientTestSuite) Test_ServiceList_Filtered() {

	util2Service, err := s.SClient.SwarmServiceInspect(context.Background(), s.Util2ID, false)
	s.Require().NoError(err)
	s.Nil(util2Service)

}

func (s *SwarmServiceClientTestSuite) Test_SwarmServiceInspect_NodeInfo_OneReplica() {
	util1Service, err := s.SClient.SwarmServiceInspect(context.Background(), s.Util1ID, true)
	s.Require().NoError(err)
	s.Require().NotNil(util1Service)

	s.Equal(s.Util1ID, util1Service.ID)
	s.Require().NotNil(util1Service.NodeInfo)

	nodeInfo := util1Service.NodeInfo
	s.Require().Len(nodeInfo, 1)
}

func (s *SwarmServiceClientTestSuite) Test_SwarmServiceInspect_NodeInfo_TwoReplica() {

	util4Service, err := s.SClient.SwarmServiceInspect(context.Background(), s.Util4ID, true)
	s.Require().NoError(err)
	s.Require().NotNil(util4Service)

	s.Equal(s.Util4ID, util4Service.ID)
	s.Require().NotNil(util4Service.NodeInfo)

	nodeInfo := util4Service.NodeInfo
	s.Require().Len(nodeInfo, 2)
}

func (s *SwarmServiceClientTestSuite) Test_SwarmServiceInspect_IncorrectName() {
	_, err := s.SClient.SwarmServiceInspect(context.Background(), "cowsfly", true)
	s.Error(err)
}

func (s *SwarmServiceClientTestSuite) Test_SwarmServiceList_GetNodeInfo() {
	services, err := s.SClient.SwarmServiceList(context.Background())
	s.Require().NoError(err)
	s.Len(services, 3)

	for _, ss := range services {
		nodeInfo, err := s.SClient.GetNodeInfo(context.Background(), ss)
		s.Require().NoError(err)
		if ss.Spec.Name == "util-1" || ss.Spec.Name == "util-4" {
			s.NotNil(nodeInfo)
		} else {
			s.Nil(nodeInfo)
		}
	}
}
func (s *SwarmServiceClientTestSuite) Test_SwarmServiceList_ServiceNamePrefix() {
	s.SClient.ServiceNamePrefix = "dev1"
	services, err := s.SClient.SwarmServiceList(context.Background())
	s.Require().NoError(err)
	s.Len(services, 3)

	expectedNames := []string{"dev1_util-1", "dev1_util-3", "dev1_util-4"}
	for _, ss := range services {
		s.Contains(expectedNames, ss.Spec.Name)
	}
}
