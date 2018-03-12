package service

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// Node utils
func createNode(name string, network string) {
	exec.Command("docker", "container", "run", "-d", "--rm",
		"--privileged", "--network", network, "--name", name,
		"--hostname", name, "docker:17.12.1-ce-dind").Output()
}

func destroyNode(name string) {
	exec.Command("docker", "container", "stop", name).Output()
}

func newTestNodeDockerClient(nodeName string) (*client.Client, error) {
	host := fmt.Sprintf("tcp://%s:2375", nodeName)
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	return client.NewClient(host, dockerAPIVersion, nil, defaultHeaders)
}

func getWorkerToken(nodeName string) string {
	args := []string{"swarm", "join-token", "worker", "-q"}
	token, _ := runDockerCommandOnNode(args, nodeName)
	return strings.TrimRight(string(token), "\n")
}
func initSwarm(nodeName string) {
	args := []string{"swarm", "init"}
	runDockerCommandOnNode(args, nodeName)
}

func joinSwarm(nodeName, rootNodeName, token string) {
	rootHost := fmt.Sprintf("%s:2377", rootNodeName)
	args := []string{"swarm", "join", "--token", token, rootHost}
	runDockerCommandOnNode(args, nodeName)
}

func getNodeID(nodeName, rootNodeName string) (string, error) {
	args := []string{"node", "inspect", nodeName, "-f", "{{ .ID }}"}
	ID, err := runDockerCommandOnNode(args, rootNodeName)
	return strings.TrimRight(string(ID), "\n"), err
}

func removeNodeFromSwarm(nodeName, rootNodeName string) {
	args := []string{"node", "rm", "--force", nodeName}
	runDockerCommandOnNode(args, rootNodeName)
}

func addLabelToNode(nodeName, label, rootNodeName string) {
	args := []string{"node", "update", "--label-add", label, nodeName}
	runDockerCommandOnNode(args, nodeName)
}

func removeLabelFromNode(nodeName, label, rootNodeName string) {
	args := []string{"node", "update", "--label-rm", label, nodeName}
	runDockerCommandOnNode(args, nodeName)
}

func runDockerCommandOnNode(args []string, nodeName string) (string, error) {
	host := fmt.Sprintf("tcp://%s:2375", nodeName)
	dockerCmd := []string{"-H", host}
	fullCmd := append(dockerCmd, args...)
	output, err := exec.Command("docker", fullCmd...).Output()
	return string(output), err
}

// Service Utils

func createTestOverlayNetwork(name string) {
	args := []string{"network", "create", "-d", "overlay", name}
	runDockerCommandOnSocket(args)
}

func removeTestNetwork(name string) {
	args := []string{"network", "create", "rm", name}
	runDockerCommandOnSocket(args)
}

func getServiceID(name string) (string, error) {
	args := []string{"service", "inspect", name, "-f", "{{ .ID }}"}
	ID, err := runDockerCommandOnSocket(args)
	return strings.TrimRight(string(ID), "\n"), err
}

func createTestService(name string, labels []string, global bool, replicas string, network string) {
	args := []string{"service", "create", "--name", name}
	for _, v := range labels {
		args = append(args, "-l", v)
	}
	if global {
		args = append(args, "--mode", "global")
	} else if len(replicas) > 0 {
		args = append(args, "--replicas", replicas)
	}
	if len(network) > 0 {
		args = append(args, "--network", network)
	}
	args = append(args, "alpine", "sleep", "1000000000")
	runDockerCommandOnSocket(args)
}

func replicaTestService(name string, count string) {
	args := []string{"service", "update", "--replicas", count, name}
	runDockerCommandOnSocket(args)
}

func removeTestService(name string) {
	args := []string{"service", "rm", name}
	runDockerCommandOnSocket(args)
}

func addLabelToService(name, label string) {
	args := []string{"service", "update", "--label-add", label, name}
	runDockerCommandOnSocket(args)
}

func removeLabelFromService(name, label string) {
	args := []string{"service", "update", "--label-rm", label, name}
	runDockerCommandOnSocket(args)
}

func getNetworkNameWithSuffix(suffix string) (string, error) {
	filter := fmt.Sprintf("name=%s$", suffix)
	args := []string{"network", "ls", "--filter", filter, "--format", "{{ .ID }}"}
	output, err := runDockerCommandOnSocket(args)
	if err != nil {
		return "", err
	}
	firstNetwork := strings.Split(output, "\n")
	return firstNetwork[0], nil
}

func runDockerCommandOnSocket(args []string) (string, error) {
	output, err := exec.Command("docker", args...).Output()
	return string(output), err
}

// SwarmServiceMini Utils

func getNewSwarmServiceMini() SwarmServiceMini {
	nodeSet := NodeIPSet{}
	nodeSet.Add("node-1", "1.0.0.1")

	return SwarmServiceMini{
		ID:   "serviceID",
		Name: "demo-go",
		Labels: map[string]string{
			"com.df.hello": "nyc",
		},
		Replicas: uint64(3),
		Global:   false,
		NodeInfo: &nodeSet,
	}
}

func getNewNodeMini() NodeMini {
	return NodeMini{
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
}
