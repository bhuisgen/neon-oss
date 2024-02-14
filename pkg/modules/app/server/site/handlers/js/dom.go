package js

import (
	"errors"
)

// domElement implement a DOM element.
type domElement struct {
	id string
	m  map[string]string
}

// newDOMElement returns a new DOM element.
func newDOMElement(id string) *domElement {
	return &domElement{
		id: id,
		m:  map[string]string{},
	}
}

// Id return the element id.
func (e *domElement) Id() string {
	return e.id
}

// Attributes returns the attributes list.
func (e *domElement) Attributes() []string {
	attributes := []string{}
	for k := range e.m {
		attributes = append(attributes, k)
	}
	return attributes
}

// GetAttribute returns the given attribute value.
func (e *domElement) GetAttribute(name string) string {
	value, ok := e.m[name]
	if !ok {
		return ""
	}
	return value
}

// SetAttribute sets the given attribute value.
func (e *domElement) SetAttribute(key string, value string) {
	e.m[key] = value
}

// domElementList implements a list of DOM elements.
type domElementList struct {
	ids []string
	m   map[string]*domElement
}

// newDOMElementList returns a new DOM element list.
func newDOMElementList() *domElementList {
	return &domElementList{
		ids: []string{},
		m:   map[string]*domElement{},
	}
}

// Ids returns the elements ids.
func (l *domElementList) Ids() []string {
	return l.ids
}

// Get returns the element with the given id.
func (l *domElementList) Get(id string) (*domElement, error) {
	e, ok := l.m[id]
	if !ok {
		return nil, errors.New("invalid id")
	}
	return e, nil
}

// Set updates the given element.
func (l *domElementList) Set(e *domElement) {
	_, ok := l.m[e.id]
	if !ok {
		l.ids = append(l.ids, e.id)
	}
	l.m[e.id] = e
}
