// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"testing"
	"time"
)

func TestMemoryCacheNew(t *testing.T) {
	type args struct {
		maxTTL  time.Duration
		maxSize int
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
	}{
		{
			name: "default",
			args: args{
				maxTTL:  0,
				maxSize: 0,
			},
		},
		{
			name: "custom values",
			args: args{
				maxTTL:  3600,
				maxSize: 10 ^ 6,
			},
		},
		{
			name: "invalid values",
			args: args{
				maxTTL:  -1,
				maxSize: -1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.maxTTL, tt.args.maxSize); got == nil {
				t.Errorf("New() got %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestMemoryCacheClose(t *testing.T) {
	maxTTL := time.Second

	cache := New(maxTTL, 0)
	cache.Set("key", "value")
	time.Sleep(maxTTL)

	if v := cache.Close(); v != nil {
		t.Errorf("failed to close got %v, want %v", v, nil)
	}
}

func TestMemoryCacheGet(t *testing.T) {
	key := "test"
	value := "value"

	cache := New(0, 0)
	cache.Set(key, value)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestMemoryCacheGet_InvalidKey(t *testing.T) {
	key := "test"
	value := "value"

	cache := New(0, 0)
	cache.Set(key, value)

	if v := cache.Get("invalid"); v != nil {
		t.Errorf("failed to get key: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheGet_ExpiredKey(t *testing.T) {
	key := "test"
	value := "value"
	ttl := time.Duration(2) * time.Second

	cache := New(0, 0)
	cache.SetWithTTL(key, value, ttl)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key1: got %v, want %v", v, value)
	}
	time.Sleep(ttl)
	if v := cache.Get(key); v != nil {
		t.Errorf("failed to get key1: got %v, want %v", v, nil)
	}

}

func TestMemoryCacheGet_EvictedKeyByMaxTTL(t *testing.T) {
	key := "test"
	value := "value"
	ttl := time.Duration(2) * time.Second

	cache := New(ttl, 0)
	cache.Set(key, value)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key1: got %v, want %v", v, value)
	}
	time.Sleep(ttl)
	if v := cache.Get(key); v != nil {
		t.Errorf("failed to get key1: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheGet_EvictedKeyBySize(t *testing.T) {
	key1 := "test1"
	key2 := "test2"
	value := "value"

	cache := New(0, 1)
	cache.Set(key1, value)
	cache.Set(key2, value)

	if v := cache.Get(key1); v != nil {
		t.Errorf("failed to get key1: got %v, want %v", v, nil)
	}
	if v := cache.Get(key2); v != value {
		t.Errorf("failed to get key2: got %v, want %v", v, value)
	}
}

func TestMemoryCacheSet(t *testing.T) {
	key := "test"
	value := "value"

	cache := New(0, 0)
	cache.Set(key, value)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestMemoryCacheSet_UpdateValue(t *testing.T) {
	key := "test"
	value1 := "value1"
	value2 := "value2"

	cache := New(0, 0)
	cache.Set(key, value1)
	cache.Set(key, value2)

	if v := cache.Get(key); v != value2 {
		t.Errorf("failed to get key: got %v, want %v", v, value2)
	}
}

func TestMemoryCacheSetWithTTL(t *testing.T) {
	key := "test"
	value := "value"
	ttl := time.Second

	cache := New(time.Minute, 0)
	cache.SetWithTTL(key, value, ttl)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to set key: got %v, want %v", v, value)
	}
	time.Sleep(ttl)
	if v := cache.Get(key); v != nil {
		t.Errorf("failed to set key: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheSetWithTTL_UpdateValue(t *testing.T) {
	key := "test"
	value1 := "value1"
	value2 := "value2"
	ttl := time.Second

	cache := New(time.Minute, 0)
	cache.SetWithTTL(key, value1, ttl)
	cache.SetWithTTL(key, value2, ttl)

	if v := cache.Get(key); v != value2 {
		t.Errorf("failed to get key: got %v, want %v", v, value2)
	}
}

func TestMemoryCacheSetWithTTL_UpdateTTL(t *testing.T) {
	key := "test"
	value := "value"
	ttl1 := time.Second
	ttl2 := time.Duration(2) * time.Second

	cache := New(time.Minute, 0)
	cache.SetWithTTL(key, value, ttl1)
	cache.SetWithTTL(key, value, ttl2)

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

func TestMemoryCacheSetWithTTL_InvalidTTL(t *testing.T) {
	key1 := "test1"
	key2 := "test2"
	value := "value"
	maxTTL := time.Second

	cache := New(maxTTL, 0)
	cache.SetWithTTL(key1, value, -1)
	cache.SetWithTTL(key2, value, time.Minute)

	if v := cache.Get(key1); v != value {
		t.Errorf("failed to set key: got %v, want %v", v, value)
	}
	if v := cache.Get(key2); v != value {
		t.Errorf("failed to set key: got %v, want %v", v, value)
	}
	time.Sleep(maxTTL)
	if v := cache.Get(key1); v != nil {
		t.Errorf("failed to set key: got %v, want %v", v, nil)
	}
	if v := cache.Get(key2); v != nil {
		t.Errorf("failed to set key: got %v, want %v", v, nil)
	}
}

func TestMemoryCacheRemove(t *testing.T) {
	key := "test"
	value := "value"

	cache := New(0, 0)
	cache.Set(key, value)
	cache.Remove(key)

	if v := cache.Get(key); v != nil {
		t.Error("failed to remove key")
	}
}

func TestMemoryCacheClear(t *testing.T) {
	key := "test"
	value := "value"

	cache := New(0, 0)
	cache.Set(key, value)
	cache.Clear()

	if v := cache.Get(key); v != nil {
		t.Error("failed to clear cache")
	}
}
