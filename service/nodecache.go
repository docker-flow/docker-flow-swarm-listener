package service

import "sync"

// NodeCacher caches sevices
type NodeCacher interface {
	InsertAndCheck(n NodeMini) bool
	IsNewOrUpdated(n NodeMini) bool
	Delete(ID string)
	Get(ID string) (NodeMini, bool)
	Keys() map[string]struct{}
}

// NodeCache implements `NodeCacher`
// Not threadsafe!
type NodeCache struct {
	cache map[string]NodeMini
	mux   sync.RWMutex
}

// NewNodeCache creates a new `NewNodeCache`
func NewNodeCache() *NodeCache {
	return &NodeCache{
		cache: map[string]NodeMini{},
	}
}

// InsertAndCheck inserts `NodeMini` into cache
// If the node is new or updated `InsertAndCheck` returns true.
func (c *NodeCache) InsertAndCheck(n NodeMini) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	cachedNode, ok := c.cache[n.ID]
	c.cache[n.ID] = n

	return !ok || !n.Equal(cachedNode)
}

// Delete removes node from cache
func (c *NodeCache) Delete(ID string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.cache, ID)
}

// Get gets node from cache
func (c *NodeCache) Get(ID string) (NodeMini, bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	v, ok := c.cache[ID]
	return v, ok
}

// IsNewOrUpdated returns true if node is new or updated
func (c *NodeCache) IsNewOrUpdated(n NodeMini) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()

	cachedNode, ok := c.cache[n.ID]
	return !ok || !n.Equal(cachedNode)
}

// Keys return the keys of the cache
func (c *NodeCache) Keys() map[string]struct{} {
	c.mux.RLock()
	defer c.mux.RUnlock()
	output := map[string]struct{}{}
	for key := range c.cache {
		output[key] = struct{}{}
	}
	return output
}
