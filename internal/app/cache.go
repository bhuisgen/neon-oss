// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"sync"
	"time"
)

// cache implements a cache of objects
type cache struct {
	items map[string]*cacheItem
	lock  sync.RWMutex
}

// cacheItem implements a cache entry
type cacheItem struct {
	Value    interface{}
	ExpireAt time.Time
}

// NewCache returns a new cache instance
func NewCache() *cache {
	return &cache{
		items: make(map[string]*cacheItem),
		lock:  sync.RWMutex{},
	}
}

// Get returns the object in the cache with the given key
func (c *cache) Get(key string) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if v, ok := c.items[key]; ok {
		if !v.ExpireAt.IsZero() && v.ExpireAt.Before(time.Now()) {
			return nil
		}

		return v.Value
	}

	return nil
}

// Set stores an object in the cache with the given key and ttl
func (c *cache) Set(key string, value interface{}, ttl time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()

	expire := time.Time{}
	if ttl != 0 {
		expire = time.Now().Add(ttl)
	}

	c.items[key] = &cacheItem{
		Value:    value,
		ExpireAt: expire,
	}
}

// Remove remove the cached object with the given key
func (c *cache) Remove(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.items, key)
}

// Clear remove all cached objects
func (c *cache) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for key := range c.items {
		delete(c.items, key)
	}
}
