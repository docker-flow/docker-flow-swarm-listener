package service

// SwarmServiceCacher caches sevices
type SwarmServiceCacher interface {
	InsertAndCheck(ss SwarmServiceMini) bool
	Delete(ID string)
	Get(ID string) (SwarmServiceMini, bool)
	Len() int
}

// SwarmServiceCache implements `SwarmServiceCacher`
// Not threadsafe!
type SwarmServiceCache struct {
	cache map[string]SwarmServiceMini
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
	cachedService, ok := c.cache[ss.ID]
	c.cache[ss.ID] = ss

	return !ok || !ss.Equal(cachedService)

}

// Delete delets service from cache
func (c *SwarmServiceCache) Delete(ID string) {
	delete(c.cache, ID)
}

// Get gets service from cache
func (c SwarmServiceCache) Get(ID string) (SwarmServiceMini, bool) {
	v, ok := c.cache[ID]
	return v, ok
}

// Len returns the number of items in cache
func (c SwarmServiceCache) Len() int {
	return len(c.cache)
}
