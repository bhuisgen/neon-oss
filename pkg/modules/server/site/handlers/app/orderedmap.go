package app

import (
	"sync"
)

// OrderedMap
type OrderedMap interface {
	Keys() []string
	Get(key string) interface{}
	Set(key string, value interface{})
	Remove(key string)
	Clear()
}

// orderedMap implements a map with ordered keys.
type orderedMap struct {
	keys []interface{}
	data map[interface{}]interface{}
	mu   sync.RWMutex
}

// newOrderedMap returns a new data map.
func newOrderedMap() *orderedMap {
	return &orderedMap{
		data: make(map[interface{}]interface{}),
		keys: []interface{}{},
	}
}

// Keys returns the ordered keys.
func (m *orderedMap) Keys() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keys
}

// Get returns the value of the given key.
func (m *orderedMap) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

// Set adds or update a value with the given key.
func (m *orderedMap) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.data[key]
	if !ok {
		m.keys = append(m.keys, key)
	}
	m.data[key] = value
}

// Remove removes the value with the given key.
func (m *orderedMap) Remove(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.data[key]
	if !ok {
		return
	}
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
	delete(m.data, key)
}

// Clear removes all values of the map.
func (m *orderedMap) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys = []interface{}{}
	for key := range m.data {
		delete(m.data, key)
	}
}
