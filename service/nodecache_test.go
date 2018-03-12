package service

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/suite"
)

type NodeCacheTestSuite struct {
	suite.Suite
	Cache *NodeCache
	NMini NodeMini
}

func TestNodeCacheUnitTestSuite(t *testing.T) {
	suite.Run(t, new(NodeCacheTestSuite))
}

func (s *NodeCacheTestSuite) SetupTest() {
	s.Cache = NewNodeCache()
	s.NMini = getNewNodeMini()
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_NewNode_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_SameLabel_ReturnsFalse() {
	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	isUpdated = s.Cache.InsertAndCheck(s.NMini)
	s.False(isUpdated)
	s.AssertInCache(s.NMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_NewNodeLabel_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	newNMini := getNewNodeMini()
	newNMini.NodeLabels["com.df.wow2"] = "yup2"

	isUpdated = s.Cache.InsertAndCheck(newNMini)
	s.True(isUpdated)
	s.AssertInCache(newNMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_UpdateNodeLabel_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	newNMini := getNewNodeMini()
	newNMini.NodeLabels["com.df.wow"] = "yup2"

	isUpdated = s.Cache.InsertAndCheck(newNMini)
	s.True(isUpdated)
	s.AssertInCache(newNMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_NewEngineLabel_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	newNMini := getNewNodeMini()
	newNMini.NodeLabels["com.df.mars"] = "far"

	isUpdated = s.Cache.InsertAndCheck(newNMini)
	s.True(isUpdated)
	s.AssertInCache(newNMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_UpdateEngineLabel_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	newNMini := getNewNodeMini()
	newNMini.NodeLabels["com.df.world"] = "flat"

	isUpdated = s.Cache.InsertAndCheck(newNMini)
	s.True(isUpdated)
	s.AssertInCache(newNMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_ChangeRole_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	newNMini := getNewNodeMini()
	newNMini.Role = swarm.NodeRoleManager

	isUpdated = s.Cache.InsertAndCheck(newNMini)
	s.True(isUpdated)
	s.AssertInCache(newNMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_ChangeState_ReturnsTrue() {

	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	newNMini := getNewNodeMini()
	newNMini.State = swarm.NodeStateDown

	isUpdated = s.Cache.InsertAndCheck(newNMini)
	s.True(isUpdated)
	s.AssertInCache(newNMini)
}

func (s *NodeCacheTestSuite) Test_InsertAndCheck_ChangeAvailability_ReturnsTrue() {

	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	newNMini := getNewNodeMini()
	newNMini.Availability = swarm.NodeAvailabilityPause

	isUpdated = s.Cache.InsertAndCheck(newNMini)
	s.True(isUpdated)
	s.AssertInCache(newNMini)
}

func (s *NodeCacheTestSuite) Test_GetAndRemove_InCache_ReturnsNodeMini_RemovesFromCache() {

	isUpdated := s.Cache.InsertAndCheck(s.NMini)
	s.True(isUpdated)
	s.AssertInCache(s.NMini)

	removedNMini, ok := s.Cache.Get(s.NMini.ID)
	s.True(ok)
	s.Cache.Delete(s.NMini.ID)
	s.AssertNotInCache(s.NMini)
	s.Equal(s.NMini, removedNMini)
}

func (s *NodeCacheTestSuite) Test_GetAndRemove_NotInCache_ReturnsFalse() {
	_, ok := s.Cache.Get(s.NMini.ID)
	s.False(ok)
}

func (s *NodeCacheTestSuite) AssertInCache(nm NodeMini) {
	ss, ok := s.Cache.Get(nm.ID)
	s.True(ok)
	s.Equal(nm, ss)
}

func (s *NodeCacheTestSuite) AssertNotInCache(nm NodeMini) {
	_, ok := s.Cache.Get(nm.ID)
	s.False(ok)
}
