// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"net/http"
)

// Fetcher
type Fetcher interface {
	Fetch(ctx context.Context, name string) error
	Exists(name string) bool
	Get(name string) ([]byte, error)
	Register(r *Resource)
	Unregister(name string)
	CreateResourceFromTemplate(template string, resource string, params map[string]string,
		headers map[string]string) (*Resource, error)
}

// Loader
type Loader interface {
	Start() error
	Stop() error
}

// LoaderExecutor
type LoaderExecutor interface {
	execute(stop <-chan struct{})
}

// Renderer
type Renderer interface {
	Handle(w http.ResponseWriter, r *http.Request, info *ServerInfo)
	Next(next Renderer)
}

// Server
type Server interface {
	Start() error
	Stop(ctx context.Context) error
}

// Render
type Render struct {
	Body           []byte
	Valid          bool
	Status         int
	Redirect       bool
	RedirectTarget string
	RedirectStatus int
	Headers        map[string]string
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

// ServerInfo
type ServerInfo struct {
	Addr    string
	Port    int
	Version string
}
