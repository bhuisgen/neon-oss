package app

import (
	"testing"
)

func TestCacheGet(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache(1)
	cache.Set(key, value)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestCacheGet_InvalidKey(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache(1)
	cache.Set(key, value)

	if v := cache.Get("invalid"); v != nil {
		t.Errorf("failed to get key: got %v, want %v", v, nil)
	}
}

func TestCacheSet(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache(1)
	cache.Set(key, value)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestCacheSet_ExistingKey(t *testing.T) {
	key := "test"
	value1 := "value1"
	value2 := "value2"

	cache := newCache(1)
	cache.Set(key, value1)
	cache.Set(key, value2)

	if v := cache.Get(key); v != value2 {
		t.Errorf("failed to get key: got %v, want %v", v, value2)
	}
}

func TestCacheSet_MaxCapacity(t *testing.T) {
	key1 := "test1"
	key2 := "test2"
	value1 := "value1"
	value2 := "value2"

	cache := newCache(1)
	cache.Set(key1, value1)
	cache.Set(key2, value2)

	if v := cache.Get(key1); v != nil {
		t.Errorf("failed to get key: got %v, want %v", v, value2)
	}
	if v := cache.Get(key2); v != value2 {
		t.Errorf("failed to get key: got %v, want %v", v, value2)
	}
}

func TestCacheRemove(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache(1)
	cache.Set(key, value)

	cache.Remove(key)
	if v := cache.Get("invalid"); v != nil {
		t.Errorf("failed to remove key: got %v, want %v", v, nil)
	}
}

func TestCacheClear(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache(1)
	cache.Set(key, value)
	cache.Clear()

	if v := cache.Get("invalid"); v != nil {
		t.Errorf("failed to clear cache: got %v, want %v", v, nil)
	}
}
