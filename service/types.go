package service

import (
	"encoding/json"

	"github.com/docker/docker/api/types/swarm"
)

// SwarmService defines internal structure with service information
type SwarmService struct {
	swarm.Service
	NodeInfo *NodeIPSet
}

// NodeIP defines a node/addr pair
type NodeIP struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
}

// NodeIPSet is a set of NodeIPs
type NodeIPSet map[NodeIP]struct{}

// Add node to set
func (ns *NodeIPSet) Add(name, addr string) {
	(*ns)[NodeIP{Name: name, Addr: addr}] = struct{}{}
}

// Equal returns true when NodeIPSets contain the same elements
func (ns NodeIPSet) Equal(other NodeIPSet) bool {

	if ns.Cardinality() != other.Cardinality() {
		return false
	}

	for ip := range ns {
		if _, ok := other[ip]; !ok {
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
	items := make([][2]string, 0, ns.Cardinality())

	for elem := range ns {
		items = append(items, [2]string{elem.Name, elem.Addr})
	}
	return json.Marshal(items)
}

// UnmarshalJSON recreates NodeIPSet from a JSON array
func (ns *NodeIPSet) UnmarshalJSON(b []byte) error {

	items := [][2]string{}
	err := json.Unmarshal(b, &items)
	if err != nil {
		return err
	}

	for _, item := range items {
		(*ns)[NodeIP{Name: item[0], Addr: item[1]}] = struct{}{}
	}

	return nil
}
