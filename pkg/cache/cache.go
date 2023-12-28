// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"sync"
	"time"
)

// Cache
type Cache interface {
	Get(key string) interface{}
	Set(key string, value interface{}, ttl time.Duration)
	Remove(key string)
	Clear()
}

// cache implements a cache of objects.
type cache struct {
	objects map[string]cacheObject
	lock    sync.RWMutex
}

// cacheObject implements an object in the cache.
type cacheObject struct {
	Value    interface{}
	ExpireAt time.Time
}

// NewCache returns a new cache.
func NewCache() *cache {
	return &cache{
		objects: make(map[string]cacheObject),
	}
}

// Get returns the value of the cached object with the given key.
func (c *cache) Get(key string) interface{} {
	c.lock.RLock()
	v, ok := c.objects[key]
	c.lock.RUnlock()

	if ok {
		if !v.ExpireAt.IsZero() && v.ExpireAt.Before(time.Now()) {
			c.lock.Lock()
			delete(c.objects, key)
			c.lock.Unlock()

			return nil
		}

		return v.Value
	}

	return nil
}

// Set stores a object in the cache with given key, value and ttl.
func (c *cache) Set(key string, value interface{}, ttl time.Duration) {
	item := cacheObject{
		Value: value,
	}
	if ttl > 0 {
		item.ExpireAt = time.Now().Add(ttl)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.objects[key] = item
}

// Remove remove the cached object with the given key.
func (c *cache) Remove(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.objects, key)
}

// Clear remove all cached objects.
func (c *cache) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for key := range c.objects {
		delete(c.objects, key)
	}
}

var _ Cache = (*cache)(nil)
