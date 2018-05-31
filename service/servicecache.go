package service

import "sync"

// SwarmServiceCacher caches sevices
type SwarmServiceCacher interface {
	InsertAndCheck(ss SwarmServiceMini) bool
	IsNewOrUpdated(ss SwarmServiceMini) bool
	Delete(ID string)
	Get(ID string) (SwarmServiceMini, bool)
	Len() int
	Keys() map[string]struct{}
}

// SwarmServiceCache implements `SwarmServiceCacher`
type SwarmServiceCache struct {
	cache map[string]SwarmServiceMini
	mux   sync.RWMutex
}

// NewSwarmServiceCache creates a new `NewSwarmServiceCache`
func NewSwarmServiceCache() *SwarmServiceCache {
	return &SwarmServiceCache{
		cache: map[string]SwarmServiceMini{},
	}
}

// InsertAndCheck inserts `SwarmServiceMini` into cache
// If the service is new or updated `InsertAndCheck` returns true.
func (c *SwarmServiceCache) InsertAndCheck(ss SwarmServiceMini) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	cachedService, ok := c.cache[ss.ID]
	c.cache[ss.ID] = ss

	return !ok || !ss.Equal(cachedService)

}

// IsNewOrUpdated returns true if service is new or updated
func (c *SwarmServiceCache) IsNewOrUpdated(ss SwarmServiceMini) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()

	cachedService, ok := c.cache[ss.ID]
	return !ok || !ss.Equal(cachedService)
}

// Delete delets service from cache
func (c *SwarmServiceCache) Delete(ID string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	delete(c.cache, ID)
}

// Get gets service from cache
func (c *SwarmServiceCache) Get(ID string) (SwarmServiceMini, bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	v, ok := c.cache[ID]
	return v, ok
}

// Len returns the number of items in cache
func (c *SwarmServiceCache) Len() int {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return len(c.cache)
}

// Keys returns the keys of the cache
func (c *SwarmServiceCache) Keys() map[string]struct{} {
	c.mux.RLock()
	defer c.mux.RUnlock()
	output := map[string]struct{}{}
	for key := range c.cache {
		output[key] = struct{}{}
	}
	return output
}
