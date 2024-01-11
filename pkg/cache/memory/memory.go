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
	m       map[string]mapValue
	q       queue
	maxTTL  time.Duration
	maxSize int
	keep    bool
	mu      sync.RWMutex
	done    chan struct{}
}

// mapValue implements a value of the map.
type mapValue struct {
	data   any
	expire time.Time
}

// queueValue implements a value of the queue.
type queueValue struct {
	key    string
	expire time.Time
}

const (
	infiniteTTL  time.Duration = time.Hour * 24 * 365 * 290
	cleanupDelay time.Duration = time.Minute * 15
)

// New returns a new memory cache.
func New(maxTTL time.Duration, maxSize int) *memoryCache {
	if maxTTL <= 0 {
		maxTTL = infiniteTTL
	}
	if maxSize < 0 {
		maxSize = 0
	}

	c := memoryCache{
		m:       make(map[string]mapValue),
		q:       newQueue(),
		maxTTL:  maxTTL,
		maxSize: maxSize,
		keep:    false,
		done:    make(chan struct{}),
	}
	heap.Init(&c.q)

	go func(done <-chan struct{}) {
		t := time.NewTicker(cleanupDelay)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				c.cleanup()
			}
		}
	}(c.done)

	return &c
}

// Close releases the cache internal resources.
func (c *memoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clear()
	c.done <- struct{}{}
	close(c.done)
	c.m = nil
	c.q = nil
	return nil
}

// Get returns the object with the given key.
func (c *memoryCache) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.get(key)
	return v
}

// Set stores a object with the given key.
func (c *memoryCache) Set(key string, value any) {
	c.mu.Lock()
	c.set(key, value, 0)
	c.mu.Unlock()
}

// Set stores a object with the given key and ttl.
func (c *memoryCache) SetWithTTL(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	c.set(key, value, ttl)
	c.mu.Unlock()
}

// Remove removes the object with the given key.
func (c *memoryCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.remove(key)
}

// Clear clears all objects.
func (c *memoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clear()
}

// get returns an object.
func (c *memoryCache) get(key string) any {
	v, ok := c.m[key]
	if !ok {
		return nil
	}
	if !v.expire.IsZero() && time.Now().After(v.expire) {
		return nil
	}
	return v.data
}

// set stores an object.
func (c *memoryCache) set(key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.maxTTL
	}
	if ttl > c.maxTTL {
		ttl = c.maxTTL
	}

	var expire time.Time
	if ttl > 0 {
		expire = time.Now().Add(ttl)
	}

	_, exists := c.m[key]
	if c.maxSize > 0 && !exists && len(c.m) >= c.maxSize {
		for c.q.Len() >= c.maxSize {
			i := heap.Pop(&c.q).(*item)
			qv := i.value.(*queueValue)
			_, ok := c.m[qv.key]
			if ok && time.Now().Before(qv.expire) {
				delete(c.m, qv.key)
			}
		}
	}

	c.m[key] = mapValue{
		data:   value,
		expire: expire,
	}
	heap.Push(&c.q, &item{
		value: &queueValue{
			key:    key,
			expire: expire,
		},
		priority: int(expire.Unix()),
	})
}

// remove removes an object.
func (c *memoryCache) remove(key string) {
	delete(c.m, key)
}

// clear removes all objects.
func (c *memoryCache) clear() {
	for key := range c.m {
		delete(c.m, key)
	}
	heap.Init(&c.q)
}

// cleanup clears all expired objects.
func (c *memoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	priority := int(time.Now().Unix())
	for c.q.Len() > 0 {
		i := heap.Pop(&c.q).(*item)
		if i.priority > priority {
			heap.Push(&c.q, i)
			break
		}
		qv := i.value.(*queueValue)
		mv, ok := c.m[qv.key]
		if ok && qv.expire == mv.expire {
			delete(c.m, qv.key)
		}
	}
}

var _ cache.Cache = (*memoryCache)(nil)
