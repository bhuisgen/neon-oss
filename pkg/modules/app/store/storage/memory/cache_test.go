package memory

import (
	"fmt"
	"testing"
)

func TestCacheNew(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newCache(); got == nil {
				t.Errorf("New() got %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestCacheGet(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache()
	cache.Set(key, value)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestCacheSet(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache()
	cache.Set(key, value)

	if v := cache.Get(key); v != value {
		t.Errorf("failed to get key: got %v, want %v", v, value)
	}
}

func TestCacheSet_UpdateValue(t *testing.T) {
	key := "test"
	value1 := "value1"
	value2 := "value2"

	cache := newCache()
	cache.Set(key, value1)
	cache.Set(key, value2)

	if v := cache.Get(key); v != value2 {
		t.Errorf("failed to get key: got %v, want %v", v, value2)
	}
}

func TestCacheRemove(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache()
	cache.Set(key, value)
	cache.Remove(key)

	if v := cache.Get(key); v != nil {
		t.Error("failed to remove key")
	}
}

func TestCacheClear(t *testing.T) {
	key := "test"
	value := "value"

	cache := newCache()
	cache.Set(key, value)
	cache.Clear()

	if v := cache.Get(key); v != nil {
		t.Error("failed to clear cache")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache2 := newCache()
	key := "test"
	value := "value"

	for n := 0; n < b.N; n++ {
		cache2.Set(key, value)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := newCache()
	key := "test"
	value := "value"
	cache.Set(key, value)

	for n := 0; n < b.N; n++ {
		cache.Get(key)
	}
}

func BenchmarkCacheSetFull(b *testing.B) {
	cache := newCache()
	key := "key"
	value := "value"

	for n := 0; n < b.N; n++ {
		for i := 1; i <= 1000; i++ {
			cache.Set(fmt.Sprint(key, i), value)
		}
	}
}

var result any

func BenchmarkCacheGetFull(b *testing.B) {
	cache := newCache()
	key := "key"
	value := "value"
	for i := 1; i <= 1000; i++ {
		cache.Set(fmt.Sprint(key, i), value)
	}

	var r any
	for n := 0; n < b.N; n++ {
		for i := 1; i <= 1000; i++ {
			r = cache.Get(fmt.Sprint(key, i))
		}
	}
	result = r
}
