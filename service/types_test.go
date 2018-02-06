package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TypesTestSuite struct {
	suite.Suite
}

func TestTypesUnitTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (s *TypesTestSuite) Test_Cardinality_DifferentElms() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-2", "1.0.1.1")
	s.Equal(2, a.Cardinality())
}

func (s *TypesTestSuite) Test_Cardinality_RepeatElems() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-1", "1.0.0.1")
	s.Equal(1, a.Cardinality())
}

func (s *TypesTestSuite) Test_NodeIPSetEqual_RepeatElems() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-1", "1.0.0.1")
	b := NodeIPSet{}
	b.Add("node-1", "1.0.0.1")
	s.True(a.Equal(b))
}

func (s *TypesTestSuite) Test_NodeIPSetEqual_LenUnequal() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-2", "1.0.1.1")
	b := NodeIPSet{}
	b.Add("node-1", "1.0.0.1")
	b.Add("node-2", "1.0.1.1")
	b.Add("node-2", "1.0.1.2")
	s.False(a.Equal(b))
}

func (s *TypesTestSuite) Test_NodeIPSetEqual_EqualSets() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-2", "1.0.1.1")
	b := NodeIPSet{}
	b.Add("node-1", "1.0.0.1")
	b.Add("node-2", "1.0.1.1")
	s.True(a.Equal(b))
}

func (s *TypesTestSuite) Test_NodeIPSetEqual_AddrNotEqual() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-2", "1.0.1.1")
	b := NodeIPSet{}
	b.Add("node-1", "1.0.0.1")
	b.Add("node-2", "1.0.1.2")
	s.False(a.Equal(b))
}

func (s *TypesTestSuite) Test_NodeIPSetEqual_NodeNameNotEqual() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-2", "1.0.1.1")
	b := NodeIPSet{}
	b.Add("node-1", "1.0.0.1")
	b.Add("node-1", "1.0.1.1")
	s.False(a.Equal(b))
}

func (s *TypesTestSuite) Test_NodeIPSetEqual_EmptySets() {
	a := NodeIPSet{}
	b := NodeIPSet{}
	s.True(a.Equal(b))
}

func (s *TypesTestSuite) Test_NodeIPSetEqual_OneEmpty() {
	a := NodeIPSet{}
	b := NodeIPSet{}
	b.Add("node-1", "1.0.0.1")
	b.Add("node-1", "1.0.1.1")
	s.False(a.Equal(b))
}

func (s *TypesTestSuite) Test_NodeIPMarshallJSON_EmptySet() {
	a := NodeIPSet{}
	b, err := json.Marshal(a)
	s.Require().NoError(err)

	s.Equal([]byte("[]"), b)
}

func (s *TypesTestSuite) Test_NodeIP_JSONCycle() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-2", "1.0.1.1")
	by, err := json.Marshal(a)
	s.Require().NoError(err)

	i := NodeIPSet{}
	err = json.Unmarshal(by, &i)

	s.True(a.Equal(i))
}

func (s *TypesTestSuite) Test_NodeIPSet_Add() {
	a := NodeIPSet{}
	a.Add("node-1", "1.0.0.1")
	a.Add("node-1", "1.0.1.1")
	b := NodeIPSet{}
	b.Add("node-1", "1.0.0.1")
	b.Add("node-1", "1.0.1.1")

	s.True(a.Equal(b))
}
