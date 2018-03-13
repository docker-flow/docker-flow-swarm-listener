package service

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SwarmServiceCacheTestSuite struct {
	suite.Suite
	Cache  *SwarmServiceCache
	SSMini SwarmServiceMini
}

func TestSwarmServiceCacheUnitTestSuite(t *testing.T) {
	suite.Run(t, new(SwarmServiceCacheTestSuite))
}

func (s *SwarmServiceCacheTestSuite) SetupTest() {
	s.Cache = NewSwarmServiceCache()
	s.SSMini = getNewSwarmServiceMini()
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_NewService_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)

	s.AssertInCache(s.SSMini)
	s.Equal(1, s.Cache.Len())
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_NewServiceGlobal_ReturnsTrue() {

	s.SSMini.Replicas = uint64(0)
	s.SSMini.Global = true
	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_SameService_ReturnsFalse() {

	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)

	newSSMini := getNewSwarmServiceMini()

	isUpdated = s.Cache.InsertAndCheck(newSSMini)
	s.False(isUpdated)
	s.AssertInCache(newSSMini)
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_NewLabel_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)

	newSSMini := getNewSwarmServiceMini()
	newSSMini.Labels["com.df.whatisthis"] = "howareyou"

	isUpdated = s.Cache.InsertAndCheck(newSSMini)
	s.True(isUpdated)
	s.AssertInCache(newSSMini)
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_NewLabel_SameKey_ReturnsTrue() {
	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)

	newSSMini := getNewSwarmServiceMini()
	newSSMini.Labels["com.df.hello"] = "sf"

	isUpdated = s.Cache.InsertAndCheck(newSSMini)
	s.True(isUpdated)
	s.AssertInCache(newSSMini)
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_ChangedReplicas_ReturnsTrue() {

	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)

	newSSMini := getNewSwarmServiceMini()
	newSSMini.Replicas = uint64(4)

	isUpdated = s.Cache.InsertAndCheck(newSSMini)
	s.True(isUpdated)
	s.AssertInCache(newSSMini)
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_ReplicasDescToZero_ReturnsTrue() {

	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)

	newSSMini := getNewSwarmServiceMini()
	newSSMini.Replicas = uint64(0)

	isUpdated = s.Cache.InsertAndCheck(newSSMini)
	s.True(isUpdated)
	s.AssertInCache(newSSMini)
}

func (s *SwarmServiceCacheTestSuite) Test_InsertAndCheck_NewNodeInfo_ReturnsTrue() {

	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)

	newSSMini := getNewSwarmServiceMini()
	nodeSet := NodeIPSet{}
	nodeSet.Add("node-3", "1.0.2.1")
	newSSMini.NodeInfo = nodeSet

	isUpdated = s.Cache.InsertAndCheck(newSSMini)
	s.True(isUpdated)
	s.AssertInCache(newSSMini)
}

func (s *SwarmServiceCacheTestSuite) Test_GetAndRemove_InCache_ReturnsSwarmServiceMini_RemovesFromCache() {

	isUpdated := s.Cache.InsertAndCheck(s.SSMini)
	s.True(isUpdated)
	s.AssertInCache(s.SSMini)

	removedSSMini, ok := s.Cache.Get(s.SSMini.ID)
	s.True(ok)
	s.Cache.Delete(s.SSMini.ID)
	s.AssertNotInCache(s.SSMini)
	s.Equal(s.SSMini, removedSSMini)

}

func (s *SwarmServiceCacheTestSuite) Test_GetAndRemove_NotInCache_ReturnsFalse() {

	_, ok := s.Cache.Get(s.SSMini.ID)
	s.False(ok)
}

func (s *SwarmServiceCacheTestSuite) AssertInCache(ssm SwarmServiceMini) {
	ss, ok := s.Cache.Get(ssm.ID)
	s.True(ok)
	s.Equal(ssm, ss)
}

func (s *SwarmServiceCacheTestSuite) AssertNotInCache(ssm SwarmServiceMini) {
	_, ok := s.Cache.Get(ssm.ID)
	s.False(ok)
}
