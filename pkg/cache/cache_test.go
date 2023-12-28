// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"testing"
	"time"
)

func TestCacheNew(t *testing.T) {
	cache := NewCache()
	if cache == nil {
		t.Error("invalid cache")
	}
	if cache != nil && cache.objects == nil {
		t.Error("invalid cache objects")
	}
}

func TestCacheGet(t *testing.T) {
	key := "key"
	value := "value"

	cache := NewCache()
	cache.objects[key] = cacheObject{
		Value: value,
	}
	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestCacheGet_InvalidKey(t *testing.T) {
	key := "key"
	value := "value"

	cache := NewCache()
	cache.objects[key] = cacheObject{
		Value: value,
	}
	if v := cache.Get("invalid"); v != nil {
		t.Errorf("failed to get key: got %v, want %v", v, nil)
	}
}

func TestCacheGet_ExpiredKey(t *testing.T) {
	key := "key"
	value := "value"
	ttl := time.Duration(10) * time.Millisecond

	cache := NewCache()
	cache.objects[key] = cacheObject{
		Value:    value,
		ExpireAt: time.Now().Add(ttl),
	}
	time.Sleep(ttl)
	if v := cache.Get(key); v == value {
		t.Errorf("failed to get key: got %v, want %v", v, nil)
	}
}

func TestCacheSet(t *testing.T) {
	key := "key"
	value := "value"

	cache := NewCache()
	cache.Set(key, value, 0)

	if v, ok := cache.objects[key]; !ok {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestCacheSet_TTL(t *testing.T) {
	key := "key"
	value := "value"
	ttl := time.Duration(4) * time.Second

	cache := NewCache()
	cache.Set(key, value, ttl)

	if v, ok := cache.objects[key]; !ok {
		t.Errorf("failed to get key: got %v, want %v", v, value)
		if v.ExpireAt.IsZero() {
			t.Errorf("failed to get ttl: got %v", v.ExpireAt)
		}
	}
}

func TestCacheRemove(t *testing.T) {
	key := "key"
	value := "value"

	cache := NewCache()
	cache.objects[key] = cacheObject{
		Value: value,
	}
	cache.Remove(key)
	if _, ok := cache.objects[key]; ok {
		t.Error("failed to remove key")
	}
}

func TestCacheClear(t *testing.T) {
	key := "key"
	value := "value"

	cache := NewCache()
	cache.objects[key] = cacheObject{
		Value: value,
	}
	cache.Clear()
	if _, ok := cache.objects[key]; ok {
		t.Error("failed to clear cache")
	}
}
