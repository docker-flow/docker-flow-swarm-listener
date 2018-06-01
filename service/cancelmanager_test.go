package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type CancelManagerTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestCancelManagerUnitTestSuite(t *testing.T) {
	suite.Run(t, new(CancelManagerTestSuite))
}

func (s *CancelManagerTestSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *CancelManagerTestSuite) Test_Add_IDEqual_CancelsContext_Returns_Context() {
	cm := NewCancelManager()
	ctx := cm.Add(s.ctx, "id1", 1)
	cm.Add(s.ctx, "id1", 2)

L:
	for {
		select {
		case <-time.After(time.Second * 5):
			s.Fail("Timeout")
			return
		case <-ctx.Done():
			break L
		}
	}

	s.Equal(int64(2), cm.v["id1"].ReqID)
}

func (s *CancelManagerTestSuite) Test_Add_IDNotExist_Returns_Context() {

	cm := NewCancelManager()
	firstCtx := cm.Add(s.ctx, "id1", 1)
	s.NotNil(firstCtx)

	s.Require().Contains(cm.v, "id1")
	s.Equal(cm.v["id1"].ReqID, int64(1))
}

func (s *CancelManagerTestSuite) Test_Delete_IDEqual_ReqIDNotEqual_DoesNothing() {
	cm := NewCancelManager()
	cm.Add(s.ctx, "id1", 1)

	s.Require().Len(cm.v, 1)

	s.False(cm.Delete("id1", 2))
	s.Require().Len(cm.v, 1)
	s.Require().Contains(cm.v, "id1")
	s.Equal(cm.v["id1"].ReqID, int64(1))
}

func (s *CancelManagerTestSuite) Test_Delete_IDEqual_ReqIDEqual_CallsCancel_RemovesFromMemory() {
	cm := NewCancelManager()
	ctx := cm.Add(s.ctx, "id1", 1)

	s.Require().Len(cm.v, 1)

	s.True(cm.Delete("id1", 1))
	s.Require().Len(cm.v, 0)

L:
	for {
		select {
		case <-time.After(time.Second * 5):
			s.Fail("Timeout")
			return
		case <-ctx.Done():
			break L
		}
	}
}

func (s *CancelManagerTestSuite) Test_Delete_IDEqual_ReqIDEqual_CntNotZero_StaysInMemory() {
	// Set startingCnt to 2
	cm := NewCancelManager()
	cm.Add(s.ctx, "id1", 1)
	s.Require().Len(cm.v, 1)
	s.Require().Contains(cm.v, "id1")

	s.True(cm.Delete("id1", 1))
	s.Require().Len(cm.v, 0)
}
