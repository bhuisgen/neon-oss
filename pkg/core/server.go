// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import (
	"net/http"
)

// Server
type Server interface {
	Name() string
	Listeners() []string
	Hosts() []string
	Store() Store
	Fetcher() Fetcher
	RegisterMiddleware(middleware func(next http.Handler) http.Handler) error
	RegisterHandler(handler http.Handler) error
}

// ServerHandlerModule
type ServerHandlerModule interface {
	Module
	Register(server Server) error
	Start() error
	Mount() error
	Unmount()
	Stop()
}

// ServerMiddlewareModule
type ServerMiddlewareModule interface {
	Module
	Register(server Server) error
	Start() error
	Mount() error
	Unmount()
	Stop()
}
