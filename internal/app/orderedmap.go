// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

// dataMap implements a map with ordered keys
type dataMap struct {
	keys []string
	data map[string]interface{}
}

// newDataMap returns a new data map
func newDataMap() *dataMap {
	return &dataMap{
		data: make(map[string]interface{}),
		keys: []string{},
	}
}

// Get returns the value of the given key
func (m *dataMap) Get(key string) (interface{}, bool) {
	v, ok := m.data[key]
	return v, ok
}

// Set adds or update a value with the given key
func (m *dataMap) Set(key string, value interface{}) {
	_, ok := m.data[key]
	if !ok {
		m.keys = append(m.keys, key)
	}
	m.data[key] = value
}

// Delete removes the value with the given key
func (m *dataMap) Delete(key string) {
	_, ok := m.data[key]
	if !ok {
		return
	}
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
		}
	}
	delete(m.data, key)
}

// Keys returns the ordered keys
func (m *dataMap) Keys() []string {
	return m.keys
}
