// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"errors"
)

// domElement implement a DOM element.
type domElement struct {
	id string
	m  *orderedMap
}

// newDOMElement returns a new DOM element.
func newDOMElement(id string) *domElement {
	return &domElement{
		id: id,
		m:  newOrderedMap(),
	}
}

// Id return the element id.
func (e *domElement) Id() string {
	return e.id
}

// Attributes returns the attributes list.
func (e *domElement) Attributes() []string {
	attributes := []string{}
	for _, k := range e.m.Keys() {
		attributes = append(attributes, k.(string))
	}
	return attributes
}

// GetAttribute returns the given attribute value.
func (e *domElement) GetAttribute(name string) string {
	value, ok := e.m.Get(name)
	if !ok {
		return ""
	}
	return value.(string)
}

// SetAttribute sets the given attribute value.
func (e *domElement) SetAttribute(key string, value string) {
	e.m.Set(key, value)
}

// domElementList implements a list of DOM elements.
type domElementList struct {
	ids  []string
	data *orderedMap
}

// newDOMElementList returns a new DOM element list.
func newDOMElementList() *domElementList {
	return &domElementList{
		ids:  []string{},
		data: newOrderedMap(),
	}
}

// Ids returns the elements ids.
func (l *domElementList) Ids() []string {
	return l.ids
}

// Get returns the element with the given id.
func (l *domElementList) Get(id string) (*domElement, error) {
	e, ok := l.data.Get(id)
	if !ok {
		return nil, errors.New("invalid id")
	}
	return e.(*domElement), nil
}

// Set updates the given element.
func (l *domElementList) Set(e *domElement) {
	_, ok := l.data.Get(e.id)
	if !ok {
		l.ids = append(l.ids, e.id)
	}
	l.data.Set(e.id, e)
}