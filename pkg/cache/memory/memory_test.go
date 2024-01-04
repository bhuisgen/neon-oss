// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"testing"
	"time"
)

func TestMemoryCacheNew(t *testing.T) {
	cache := NewMemoryCache()
	if cache == nil {
		t.Error("invalid cache")
	}
}

func TestMemoryCacheGet(t *testing.T) {
	key := "test"
	value := "value"

	cache := NewMemoryCache()
	cache.Set(key, value, 0)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestMemoryCacheGet_InvalidKey(t *testing.T) {
	key := "test"
	value := "value"

	cache := NewMemoryCache()
	cache.Set(key, value, 0)

	if v := cache.Get("invalid"); v != nil {
		t.Errorf("failed to get key: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheGet_ExpiredKey(t *testing.T) {
	key1 := "test1"
	key2 := "test2"
	value := "value"
	ttl1 := time.Duration(1) * time.Second
	ttl2 := time.Duration(2) * time.Second

	cache := NewMemoryCache()
	cache.Set(key1, value, ttl1)
	cache.Set(key2, value, ttl2)

	if v := cache.Get(key1); v != value {
		t.Errorf("failed to get key1: got %v, want %v", v, value)
	}
	if v := cache.Get(key2); v != value {
		t.Errorf("failed to get key2: got %v, want %v", v, value)
	}
	time.Sleep(ttl1)
	if v := cache.Get(key1); v != nil {
		t.Errorf("failed to get key1: got %v, want %v", v, nil)
	}
	if v := cache.Get(key2); v != value {
		t.Errorf("failed to get key2: got %v, want %v", v, value)
	}
	time.Sleep(ttl2)
	if v := cache.Get(key2); v != nil {
		t.Errorf("failed to get key2: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheSet(t *testing.T) {
	key := "test"
	value := "value"

	cache := NewMemoryCache()
	cache.Set(key, value, 0)

	if v, ok := cache.m[key]; !ok {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestMemoryCacheSet_TTL(t *testing.T) {
	key := "test"
	value := "value"
	ttl := time.Duration(1) * time.Second

	cache := NewMemoryCache()
	cache.Set(key, value, ttl)

	if v, ok := cache.m[key]; !ok {
		t.Errorf("failed to set key: got %v, want %v", v, value)
	}
	time.Sleep(ttl)
	if v := cache.Get(key); v != nil {
		t.Errorf("failed to set key: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheSet_UpdateValue(t *testing.T) {
	key := "test"
	value1 := "value1"
	value2 := "value2"

	cache := NewMemoryCache()
	cache.Set(key, value1, 0)
	cache.Set(key, value2, 0)

	if v := cache.Get(key); v != value2 {
		t.Errorf("failed to get key: got %v, want %v", v, value2)
	}
}

func TestMemoryCacheSet_UpdateTTL(t *testing.T) {
	key := "test"
	value := "value"
	ttl1 := time.Duration(1) * time.Second
	ttl2 := time.Duration(2) * time.Second

	cache := NewMemoryCache()
	cache.Set(key, value, ttl1)
	cache.Set(key, value, ttl2)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
	time.Sleep(ttl1)
	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
	time.Sleep(ttl2)
	if v := cache.Get(key); v != nil {
		t.Errorf("failed to get key: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheRemove(t *testing.T) {
	key := "test"
	value := "value"

	cache := NewMemoryCache()
	cache.Set(key, value, 0)
	cache.Remove(key)

	if _, ok := cache.m[key]; ok {
		t.Error("failed to remove key")
	}
}

func TestMemoryCacheClear(t *testing.T) {
	key := "test"
	value := "value"
	ttl := time.Duration(1) * time.Second

	cache := NewMemoryCache()
	cache.Set(key, value, ttl)
	cache.Clear()

	if _, ok := cache.m[key]; ok {
		t.Error("failed to clear cache")
	}
}
