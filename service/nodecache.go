package service

// NodeCacher caches sevices
type NodeCacher interface {
	InsertAndCheck(n NodeMini) bool
	Delete(ID string)
	Get(ID string) (NodeMini, bool)
}

// NodeCache implements `NodeCacher`
// Not threadsafe!
type NodeCache struct {
	cache map[string]NodeMini
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
	cachedNode, ok := c.cache[n.ID]
	c.cache[n.ID] = n

	return !ok || !n.Equal(cachedNode)
}

// Delete removes node from cache
func (c *NodeCache) Delete(ID string) {
	delete(c.cache, ID)
}

// Get gets node from cache
func (c NodeCache) Get(ID string) (NodeMini, bool) {
	v, ok := c.cache[ID]
	return v, ok
}
