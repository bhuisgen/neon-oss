// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import "net/http"

// Renderer
type Renderer interface {
	handle(http.ResponseWriter, *http.Request)
	setNext(Renderer)
}

// Render
type Render struct {
	Body           []byte
	Valid          bool
	Status         int
	Redirect       bool
	RedirectTarget string
	RedirectStatus int
	Cache          bool
}

// Resource
type Resource struct {
	Name    string
	Method  string
	URL     string
	Params  map[string]string
	Headers map[string]string
	TTL     int64
}
