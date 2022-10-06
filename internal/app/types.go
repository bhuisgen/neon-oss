// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import "net/http"

// Renderer
type Renderer interface {
	handle(http.ResponseWriter, *http.Request, *ServerInfo)
	setNext(Renderer)
}

// Render
type Render struct {
	Body           []byte `json:"body"`
	Valid          bool   `json:"valid"`
	Status         int    `json:"status"`
	Redirect       bool   `json:"redirect"`
	RedirectTarget string `json:"redirectTarget"`
	RedirectStatus int    `json:"redirectStatus"`
	Cache          bool   `json:"cache"`
}

// Resource
type Resource struct {
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Params  map[string]string `json:"params"`
	Headers map[string]string `json:"headers"`
	TTL     int64             `json:"ttl"`
}

// ResourceResult
type ResourceResult struct {
	Loading  bool   `json:"loading"`
	Error    string `json:"error"`
	Response string `json:"response"`
}

// ServerInfo
type ServerInfo struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	Version string `json:"version"`
}
