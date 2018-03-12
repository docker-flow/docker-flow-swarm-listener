package service

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// NodeInspector is able to inspect a swarm node
type NodeInspector interface {
	NodeInspect(nodeID string) (swarm.Node, error)
	NodeList(ctx context.Context) ([]swarm.Node, error)
}

// NodeClient implementes `NodeInspector` for docker
type NodeClient struct {
	DockerClient *client.Client
}

// NewNodeClient creates a `NodeClient`
func NewNodeClient(c *client.Client) *NodeClient {
	return &NodeClient{DockerClient: c}
}

// NodeInspect returns `swarm.Node` from its ID
func (c NodeClient) NodeInspect(nodeID string) (swarm.Node, error) {
	node, _, err := c.DockerClient.NodeInspectWithRaw(context.Background(), nodeID)
	return node, err
}

// NodeList returns a list of all nodes
func (c NodeClient) NodeList(ctx context.Context) ([]swarm.Node, error) {
	return c.DockerClient.NodeList(ctx, types.NodeListOptions{})
}
