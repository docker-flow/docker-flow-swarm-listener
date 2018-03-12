package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type NodeInspectorTestSuite struct {
	suite.Suite
	NClient *NodeClient
}

func TestNodeInspectorTestSuite(t *testing.T) {
	suite.Run(t, new(NodeInspectorTestSuite))
}

func (s *NodeInspectorTestSuite) SetupSuite() {
	c, err := newTestNodeDockerClient("node1")
	s.Require().NoError(err)
	s.NClient = NewNodeClient(c)

	// Create swarm of two nodes
	// Assumes running test with docker-compose.yml
	network, err := getNetworkNameWithSuffix("dfsl_network")
	s.Require().NoError(err)

	createNode("node1", network)
	time.Sleep(time.Second)

	initSwarm("node1")
	time.Sleep(time.Second)

	joinToken := getWorkerToken("node1")

	createNode("node2", network)
	time.Sleep(time.Second)
	joinSwarm("node2", "node1", joinToken)
	time.Sleep(time.Second)
}

func (s *NodeInspectorTestSuite) TearDownSuite() {
	destroyNode("node2")
	destroyNode("node1")
}

func (s *NodeInspectorTestSuite) Test_NodeInspect() {
	nodeID, err := getNodeID("node2", "node1")
	s.Require().NoError(err)

	node, err := s.NClient.NodeInspect("node2")
	s.Require().NoError(err)

	s.Equal("node2", node.Description.Hostname)
	s.Equal(nodeID, node.ID)
}

func (s *NodeInspectorTestSuite) Test_NodeInspect_Error() {

	_, err := s.NClient.NodeInspect("node3")
	s.Error(err)
}

func (s *NodeInspectorTestSuite) Test_NodeList() {
	nodes, err := s.NClient.NodeList(context.Background())
	s.Require().NoError(err)
	s.Len(nodes, 2)
}
