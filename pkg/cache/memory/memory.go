// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"container/heap"
	"sync"
	"time"

	"github.com/bhuisgen/neon/pkg/cache"
)

// memoryCache implements a memory cache.
type memoryCache struct {
	m map[string]mapValue
	q queue
	l sync.RWMutex
}

// mapValue implements a value of the map.
type mapValue struct {
	data    any
	version int
}

// queueValue implements a value of the queue.
type queueValue struct {
	key     string
	version int
}

// NewMemoryCache returns a new memory cache.
func NewMemoryCache() *memoryCache {
	c := memoryCache{
		m: make(map[string]mapValue),
		q: newQueue(),
	}
	heap.Init(&c.q)
	return &c
}

// Get returns the object with the given key.
func (c *memoryCache) Get(key string) any {
	c.l.Lock()
	c.clearExpired(int(time.Now().Unix()))
	c.l.Unlock()

	c.l.RLock()
	v, _ := c.get(key)
	c.l.RUnlock()
	return v
}

// Set stores a object with the given key, value and ttl.
func (c *memoryCache) Set(key string, value any, ttl time.Duration) {
	c.l.Lock()
	c.set(key, value, ttl)
	c.l.Unlock()
}

// Remove removes the object with the given key.
func (c *memoryCache) Remove(key string) {
	c.l.Lock()
	c.remove(key)
	c.l.Unlock()
}

// Clear clears all objects.
func (c *memoryCache) Clear() {
	c.l.Lock()
	c.clear()
	c.l.Unlock()
}

// get returns an object.
func (c *memoryCache) get(key string) (any, int) {
	v, ok := c.m[key]
	if !ok {
		return nil, 0
	}
	return v.data, v.version
}

// set stores an object.
func (c *memoryCache) set(key string, value any, ttl time.Duration) {
	_, oldVersion := c.get(key)
	c.m[key] = mapValue{
		data:    value,
		version: oldVersion + 1,
	}
	if ttl > 0 {
		heap.Push(&c.q, &item{
			value: &queueValue{
				key:     key,
				version: oldVersion + 1,
			},
			priority: int(time.Now().Add(ttl).Unix()),
		})
	}
}

// clearExpired clears all expired objects.
func (c *memoryCache) clearExpired(priority int) {
	for {
		if c.q.Len() == 0 {
			break
		}
		i := heap.Pop(&c.q).(*item)
		if i.priority > priority {
			heap.Push(&c.q, i)
			break
		}
		qv := i.value.(*queueValue)
		mv, ok := c.m[qv.key]
		if ok && qv.version == mv.version {
			delete(c.m, qv.key)
		}
	}
}

// remove removes an object.
func (c *memoryCache) remove(key string) {
	delete(c.m, key)
}

// clear removes all objects.
func (c *memoryCache) clear() {
	for i := 0; i < c.q.Len(); i++ {
		heap.Remove(&c.q, i)
	}
	for key := range c.m {
		delete(c.m, key)
	}
}

var _ cache.Cache = (*memoryCache)(nil)
