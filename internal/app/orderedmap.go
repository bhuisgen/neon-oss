// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

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

// orderedMap implements a map with ordered keys
type orderedMap struct {
	keys []interface{}
	data map[interface{}]interface{}
	lock sync.RWMutex
	}

// newOrderedMap returns a new data map
func newOrderedMap() *orderedMap {
	return &orderedMap{
		data: make(map[interface{}]interface{}),
		keys: []interface{}{},
	}
}

// Keys returns the ordered keys
func (m *orderedMap) Keys() []interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.keys
}

// Get returns the value of the given key
func (m *orderedMap) Get(key string) (interface{}, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	v, ok := m.data[key]
	return v, ok
}

// Set adds or update a value with the given key
func (m *orderedMap) Set(key string, value interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.data[key]
	if !ok {
		m.keys = append(m.keys, key)
	}
	m.data[key] = value
}

// Remove removes the value with the given key
func (m *orderedMap) Remove(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()

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

// Clear removes all values of the map
func (m *orderedMap) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.keys = []interface{}{}
	for key := range m.data {
		delete(m.data, key)
	}
}
