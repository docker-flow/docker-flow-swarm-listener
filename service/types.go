package service

import (
	"encoding/json"

	"github.com/docker/docker/api/types/swarm"
)

// SwarmServiceMini is a optimized version of `SwarmService` for caching purposes
type SwarmServiceMini struct {
	ID             string
	Name           string
	Labels         map[string]string
	Global         bool
	Replicas       uint64
	ContainerImage string
	NodeInfo       NodeIPSet
}

// Equal returns when SwarmServiceMini is equal to `other`
func (ssm SwarmServiceMini) Equal(other SwarmServiceMini) bool {

	return (ssm.ID == other.ID) &&
		(ssm.Name == other.Name) &&
		EqualMapStringString(ssm.Labels, other.Labels) &&
		(ssm.Global == other.Global) &&
		(ssm.Replicas == other.Replicas) &&
		(ssm.ContainerImage == other.ContainerImage) &&
		EqualNodeIPSet(ssm.NodeInfo, other.NodeInfo)
}

// NodeMini is a optimized version of `swarm.Node` for caching purposes
type NodeMini struct {
	ID           string
	Hostname     string
	VersionIndex uint64
	State        swarm.NodeState
	Addr         string
	NodeLabels   map[string]string
	EngineLabels map[string]string
	Role         swarm.NodeRole
	Availability swarm.NodeAvailability
}

// Equal returns true when NodeMini is equal to `other`
func (ns NodeMini) Equal(other NodeMini) bool {
	return (ns.ID == other.ID) &&
		(ns.Hostname == other.Hostname) &&
		(ns.State == other.State) &&
		(ns.Addr == other.Addr) &&
		EqualMapStringString(ns.NodeLabels, other.NodeLabels) &&
		EqualMapStringString(ns.EngineLabels, other.EngineLabels) &&
		(ns.Role == other.Role) &&
		(ns.Availability == other.Availability)
}

// EqualMapStringString Returns true when the two maps are equal
func EqualMapStringString(l map[string]string, r map[string]string) bool {
	if len(l) != len(r) {
		return false
	}
	for lk, lv := range l {
		if rv, ok := r[lk]; !ok || lv != rv {
			return false
		}
	}

	return true
}

// SwarmService defines internal structure with service information
type SwarmService struct {
	swarm.Service
	NodeInfo NodeIPSet
}

// EventType is the type of event from eventlisteners
type EventType string

const (
	// EventTypeCreate is for create or update event
	EventTypeCreate EventType = "create"
	// EventTypeRemove is for remove events
	EventTypeRemove EventType = "remove"
)

// Event contains information about docker events
type Event struct {
	Type     EventType
	ID       string
	TimeNano int64
	UseCache bool
}

// NodeIP defines a node/addr pair
type NodeIP struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	ID   string `json:"id"`
}

// NodeIPSet is a set of NodeIPs
type NodeIPSet map[NodeIP]struct{}

// Add node to set
func (ns NodeIPSet) Add(name, addr, id string) {
	ns[NodeIP{Name: name, Addr: addr, ID: id}] = struct{}{}
}

// EqualNodeIPSet returns true when NodeIPSets contain the same elements
func EqualNodeIPSet(l NodeIPSet, r NodeIPSet) bool {

	if l == nil && r == nil {
		return true
	} else if l == nil && r != nil {
		return false
	} else if l != nil && r == nil {
		return false
	}

	if l.Cardinality() != r.Cardinality() {
		return false
	}

	for ip := range l {
		if _, ok := (r)[ip]; !ok {
			return false
		}
	}
	return true
}

// Cardinality returns the size of set
func (ns NodeIPSet) Cardinality() int {
	return len(ns)
}

// MarshalJSON creates JSON array from NodeIPSet
func (ns NodeIPSet) MarshalJSON() ([]byte, error) {
	items := make([][]string, 0, ns.Cardinality())

	for elem := range ns {
		items = append(items, []string{elem.Name, elem.Addr, elem.ID})
	}
	return json.Marshal(items)
}

// UnmarshalJSON recreates NodeIPSet from a JSON array
func (ns *NodeIPSet) UnmarshalJSON(b []byte) error {

	items := [][]string{}
	err := json.Unmarshal(b, &items)
	if err != nil {
		return err
	}

	for _, item := range items {
		nodeIP := NodeIP{Name: item[0], Addr: item[1]}
		if len(item) == 3 {
			nodeIP.ID = item[2]
		}
		(*ns)[nodeIP] = struct{}{}
	}

	return nil
}
