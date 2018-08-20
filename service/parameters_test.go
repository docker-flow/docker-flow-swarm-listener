package service

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/suite"
)

type ParametersTestSuite struct {
	suite.Suite
}

func TestParametersUnitTestSuite(t *testing.T) {
	suite.Run(t, new(ParametersTestSuite))
}

func (s *ParametersTestSuite) Test_GetNodeMiniCreateParameters_DistributeUndefined_AddsDistrubte() {
	nm := getNewNodeMini()

	expected := map[string]string{
		"id":           "nodeID",
		"hostname":     "nodehostname",
		"address":      "nodeaddr",
		"versionIndex": "10",
		"state":        "ready",
		"role":         "worker",
		"availability": "active",
		"world":        "round",
		"wow":          "yup",
	}

	params := GetNodeMiniCreateParameters(nm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetNodeMiniCreateParameters_LabelsTakeSecondPriority() {
	nm := getNewNodeMini()
	nm.NodeLabels["com.df.state"] = "cow"
	nm.Role = swarm.NodeRoleManager
	nm.Availability = swarm.NodeAvailabilityDrain

	expected := map[string]string{
		"id":           "nodeID",
		"hostname":     "nodehostname",
		"address":      "nodeaddr",
		"versionIndex": "10",
		"state":        "ready",
		"role":         "manager",
		"availability": "drain",
		"world":        "round",
		"wow":          "yup",
	}
	params := GetNodeMiniCreateParameters(nm)
	s.Equal(expected, params)

}

func (s *ParametersTestSuite) Test_GetNodeMiniCreateParameters_NodeLabelsHigherPriority() {
	nm := getNewNodeMini()
	nm.NodeLabels["com.df.dogs"] = "chase"
	nm.EngineLabels["com.df.dogs"] = "cry"

	expected := map[string]string{
		"id":           "nodeID",
		"hostname":     "nodehostname",
		"address":      "nodeaddr",
		"versionIndex": "10",
		"state":        "ready",
		"role":         "worker",
		"availability": "active",
		"world":        "round",
		"wow":          "yup",
		"dogs":         "chase",
	}

	params := GetNodeMiniCreateParameters(nm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetNodeMiniRemoveParameters() {
	nm := getNewNodeMini()

	expected := map[string]string{
		"id":       "nodeID",
		"hostname": "nodehostname",
		"address":  "nodeaddr",
	}
	params := GetNodeMiniRemoveParameters(nm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_Global() {
	ssm := getNewSwarmServiceMini()
	ssm.Replicas = uint64(0)
	ssm.Global = true

	b, err := json.Marshal(ssm.NodeInfo)
	s.Require().NoError(err)

	expected := map[string]string{
		"serviceName": "demo-go",
		"hello":       "nyc",
		"distribute":  "true",
		"nodeInfo":    string(b),
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_LabelsTakeSecondPriority() {
	ssm := getNewSwarmServiceMini()
	ssm.Labels["com.df.serviceName"] = "thisisbad"

	b, err := json.Marshal(ssm.NodeInfo)
	s.Require().NoError(err)

	expected := map[string]string{
		"serviceName": "demo-go",
		"hello":       "nyc",
		"distribute":  "true",
		"nodeInfo":    string(b),
		"replicas":    "3",
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_Replicas() {
	ssm := getNewSwarmServiceMini()

	b, err := json.Marshal(ssm.NodeInfo)
	s.Require().NoError(err)

	expected := map[string]string{
		"serviceName": "demo-go",
		"hello":       "nyc",
		"distribute":  "true",
		"nodeInfo":    string(b),
		"replicas":    "3",
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_DistributeDefined() {
	ssm := getNewSwarmServiceMini()
	ssm.Labels["com.df.distribute"] = "false"

	b, err := json.Marshal(ssm.NodeInfo)
	s.Require().NoError(err)

	expected := map[string]string{
		"serviceName": "demo-go",
		"hello":       "nyc",
		"nodeInfo":    string(b),
		"distribute":  "false",
		"replicas":    "3",
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)

}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_NoNodeInfo() {
	ssm := getNewSwarmServiceMini()
	ssm.NodeInfo = nil

	expected := map[string]string{
		"serviceName": "demo-go",
		"hello":       "nyc",
		"distribute":  "true",
		"replicas":    "3",
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)

}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_StackNamespace_ShortNameTrue_Combines_ServiceName() {
	ssm := getNewSwarmServiceMini()
	ssm.Name = "stack_demo-go"
	ssm.Labels["com.docker.stack.namespace"] = "stack"
	ssm.Labels["com.df.shortName"] = "true"

	b, err := json.Marshal(ssm.NodeInfo)
	s.Require().NoError(err)

	expected := map[string]string{
		"serviceName": "demo-go",
		"hello":       "nyc",
		"nodeInfo":    string(b),
		"distribute":  "true",
		"replicas":    "3",
		"shortName":   "true",
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_StackNamespace_ShortNameUndefined_DoesNotCombineServiceName() {
	ssm := getNewSwarmServiceMini()
	ssm.Name = "stack_demo-go"
	ssm.Labels["com.docker.stack.namespace"] = "stack"

	b, err := json.Marshal(ssm.NodeInfo)
	s.Require().NoError(err)

	expected := map[string]string{
		"serviceName": "stack_demo-go",
		"hello":       "nyc",
		"nodeInfo":    string(b),
		"distribute":  "true",
		"replicas":    "3",
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniCreateParameters_StackNamespace_ShortNameFalse_DoesNotCombineServiceName() {
	ssm := getNewSwarmServiceMini()
	ssm.Name = "stack_demo-go"
	ssm.Labels["com.docker.stack.namespace"] = "stack"
	ssm.Labels["com.df.shortName"] = "false"

	b, err := json.Marshal(ssm.NodeInfo)
	s.Require().NoError(err)

	expected := map[string]string{
		"serviceName": "stack_demo-go",
		"hello":       "nyc",
		"nodeInfo":    string(b),
		"distribute":  "true",
		"replicas":    "3",
		"shortName":   "false",
	}

	params := GetSwarmServiceMiniCreateParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniRemoveParameters() {
	ssm := getNewSwarmServiceMini()
	expected := map[string]string{
		"serviceName": "demo-go",
		"distribute":  "true",
		"hello":       "nyc",
	}
	params := GetSwarmServiceMiniRemoveParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniRemoveParameters_ShortNameUndefined_DoesNotCombineServiceName() {
	ssm := getNewSwarmServiceMini()
	ssm.Name = "stack_demo-go"
	ssm.Labels["com.docker.stack.namespace"] = "stack"

	expected := map[string]string{
		"serviceName": "stack_demo-go",
		"distribute":  "true",
		"hello":       "nyc",
	}
	params := GetSwarmServiceMiniRemoveParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniRemoveParameters_ShortNameFalse_DoesNotCombineServiceName() {
	ssm := getNewSwarmServiceMini()
	ssm.Name = "stack_demo-go"
	ssm.Labels["com.docker.stack.namespace"] = "stack"
	ssm.Labels["com.df.shortName"] = "false"

	expected := map[string]string{
		"serviceName": "stack_demo-go",
		"distribute":  "true",
		"hello":       "nyc",
		"shortName":   "false",
	}
	params := GetSwarmServiceMiniRemoveParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_GetSwarmServiceMiniRemoveParameters_ShortNameTrue_CombineServiceName() {
	ssm := getNewSwarmServiceMini()
	ssm.Name = "stack_demo-go"
	ssm.Labels["com.docker.stack.namespace"] = "stack"
	ssm.Labels["com.df.shortName"] = "true"

	expected := map[string]string{
		"serviceName": "demo-go",
		"distribute":  "true",
		"hello":       "nyc",
		"shortName":   "true",
	}
	params := GetSwarmServiceMiniRemoveParameters(ssm)
	s.Equal(expected, params)
}

func (s *ParametersTestSuite) Test_ConvertMapStringStringToURLValues() {
	expected := url.Values{}
	expected.Add("id", "nodeID")
	expected.Add("hostname", "nodehostname")
	expected.Add("versionIndex", "10")
	expected.Add("state", "ready")
	expected.Add("role", "worker")
	expected.Add("availability", "active")

	// labels
	expected.Add("world", "round")
	expected.Add("wow", "yup")

	params := map[string]string{
		"id":           "nodeID",
		"hostname":     "nodehostname",
		"versionIndex": "10",
		"state":        "ready",
		"role":         "worker",
		"availability": "active",
		"world":        "round",
		"wow":          "yup",
	}

	convertedURLValues := ConvertMapStringStringToURLValues(params)

	for k := range params {
		s.Equal(expected.Get(k),
			convertedURLValues.Get(k))
	}

}
