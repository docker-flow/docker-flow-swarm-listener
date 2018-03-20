package service

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/suite"
)

type MinifyUnitTestSuite struct {
	suite.Suite
}

func TestMinifyUnitTest(t *testing.T) {
	suite.Run(t, new(MinifyUnitTestSuite))
}

func (s *MinifyUnitTestSuite) Test_MinifyNode() {
	meta := swarm.Meta{
		Version: swarm.Version{Index: uint64(10)},
	}
	annot := swarm.Annotations{
		Labels: map[string]string{
			"cows":       "moo",
			"birds":      "fly",
			"com.df.wow": "yup",
		},
	}
	spec := swarm.NodeSpec{
		Annotations:  annot,
		Role:         swarm.NodeRoleWorker,
		Availability: swarm.NodeAvailabilityActive,
	}
	engineDesp := swarm.EngineDescription{
		Labels: map[string]string{
			"squrriels":    "climb",
			"com.df.world": "round",
		},
	}
	des := swarm.NodeDescription{
		Hostname: "nodehostname",
		Engine:   engineDesp,
	}
	nodeStatus := swarm.NodeStatus{
		State: swarm.NodeStateReady,
		Addr:  "nodeaddr",
	}

	n := swarm.Node{
		ID:          "nodeID",
		Meta:        meta,
		Spec:        spec,
		Description: des,
		Status:      nodeStatus,
	}
	expectMini := NodeMini{
		ID:           "nodeID",
		Hostname:     "nodehostname",
		VersionIndex: uint64(10),
		State:        swarm.NodeStateReady,
		Addr:         "nodeaddr",
		NodeLabels: map[string]string{
			"com.df.wow": "yup",
		},
		EngineLabels: map[string]string{
			"com.df.world": "round",
		},
		Role:         swarm.NodeRoleWorker,
		Availability: swarm.NodeAvailabilityActive,
	}

	nodeMini := MinifyNode(n)

	s.Equal(expectMini, nodeMini)
}

func (s *MinifyUnitTestSuite) Test_MinifySwarmService_Global() {
	annot := swarm.Annotations{
		Name: "serviceName",
		Labels: map[string]string{
			"cows":          "moo",
			"birds":         "fly",
			"com.df.hello":  "nyc",
			"com.df.notify": "true",
		},
	}
	mode := swarm.ServiceMode{
		Global: &swarm.GlobalService{},
	}

	serviceSpec := swarm.ServiceSpec{
		Annotations: annot,
		Mode:        mode,
	}

	nodeSet := NodeIPSet{}
	nodeSet.Add("node-1", "1.0.0.1", "id1")
	nodeSet.Add("node-2", "1.0.1.1", "id2")

	service := swarm.Service{
		ID:   "serviceID",
		Spec: serviceSpec,
	}

	expectMini := SwarmServiceMini{
		ID:   "serviceID",
		Name: "serviceName",
		Labels: map[string]string{
			"com.df.hello": "nyc",
		},
		Global:   true,
		Replicas: uint64(0),
		NodeInfo: nodeSet,
	}

	ss := SwarmService{service, nodeSet}
	ssMini := MinifySwarmService(ss, "com.df.notify", "com.docker.stack.namespace")

	s.Equal(expectMini, ssMini)
}

func (s *MinifyUnitTestSuite) Test_MinifySwarmService_Replicas() {
	annot := swarm.Annotations{
		Name: "serviceName",
		Labels: map[string]string{
			"cows":                       "moo",
			"birds":                      "fly",
			"com.df.hello":               "world",
			"com.df.notify":              "true",
			"com.docker.stack.namespace": "really",
		},
	}
	replicas := uint64(3)
	mode := swarm.ServiceMode{
		Replicated: &swarm.ReplicatedService{
			Replicas: &replicas,
		},
	}

	serviceSpec := swarm.ServiceSpec{
		Annotations: annot,
		Mode:        mode,
	}

	nodeSet := NodeIPSet{}
	nodeSet.Add("node-1", "1.0.0.1", "id1")

	service := swarm.Service{
		ID:   "serviceID",
		Spec: serviceSpec,
	}

	expectMini := SwarmServiceMini{
		ID:   "serviceID",
		Name: "serviceName",
		Labels: map[string]string{
			"com.df.hello":               "world",
			"com.docker.stack.namespace": "really",
		},
		Global:   false,
		Replicas: uint64(3),
		NodeInfo: nodeSet,
	}

	ss := SwarmService{service, nodeSet}
	ssMini := MinifySwarmService(ss, "com.df.notify", "com.docker.stack.namespace")

	s.Equal(expectMini, ssMini)
}
