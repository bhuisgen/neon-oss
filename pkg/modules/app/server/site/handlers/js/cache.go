package js

import (
	"container/list"
	"sync"
)

// Cache
type Cache interface {
	Get(key string) any
	Set(key string, value any)
	Remove(key string)
	Clear()
}

// cache implements a LRU cache.
type cache struct {
	capacity int
	m        map[string]*cacheItem
	l        *list.List
	mu       sync.RWMutex
}

// item implements the value in the map.
type cacheItem struct {
	v any
	e *list.Element
}

// newCache creates a new cache instance.
func newCache(capacity int) *cache {
	return &cache{
		capacity: capacity,
		m:        make(map[string]*cacheItem, capacity),
		l:        list.New(),
	}
}

// Get returns the object with the given key.
func (c *cache) Get(key string) any {
	c.mu.RLock()
	if i, ok := c.m[key]; ok {
		c.l.MoveToFront(i.e)
		c.mu.RUnlock()
		return i.v
	} else {
		c.mu.RUnlock()
		return nil
	}
}

// Set stores a object with the given key.
func (c *cache) Set(key string, value any) {
	c.mu.Lock()
	if i, ok := c.m[key]; ok {
		i.v = value
		c.l.MoveToFront(i.e)
		c.m[key] = i
	} else {
		if c.l.Len() >= c.capacity {
			value := c.l.Remove(c.l.Back())
			delete(c.m, value.(string))
		}
		e := c.l.PushFront(key)
		c.m[key] = &cacheItem{
			v: value,
			e: e,
		}
	}
	c.mu.Unlock()
}

// Remove removes the object with the given key.
func (c *cache) Remove(key string) {
	c.mu.Lock()
	if i, ok := c.m[key]; ok {
		c.l.Remove(i.e)
		delete(c.m, key)
	}
	c.mu.Unlock()
}

// Clear clears all objects.
func (c *cache) Clear() {
	c.mu.Lock()
	for key, node := range c.m {
		c.l.Remove(node.e)
		delete(c.m, key)
	}
	c.mu.Unlock()
}

var _ Cache = (*cache)(nil)
