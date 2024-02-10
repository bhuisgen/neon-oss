package memory

import (
	"sync"
)

// Cache
type Cache interface {
	Get(key string) any
	Set(key string, value any)
	Remove(key string)
	Clear()
}

// cache implements a memory cache.
type cache struct {
	m  map[string]any
	mu sync.RWMutex
}

// newCache creates a new cache instance.
func newCache() *cache {
	return &cache{
		m: make(map[string]any),
	}
}

// Get returns the object with the given key.
func (c *cache) Get(key string) any {
	c.mu.RLock()
	if v, ok := c.m[key]; ok {
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()
	return nil
}

// Set stores a object with the given key.
func (c *cache) Set(key string, value any) {
	c.mu.Lock()
	c.m[key] = value
	c.mu.Unlock()
}

// Remove removes the object with the given key.
func (c *cache) Remove(key string) {
	c.mu.Lock()
	delete(c.m, key)
	c.mu.Unlock()
}

// Clear clears all objects.
func (c *cache) Clear() {
	c.mu.Lock()
	for key := range c.m {
		delete(c.m, key)
	}
	c.mu.Unlock()
}

var _ Cache = (*cache)(nil)
