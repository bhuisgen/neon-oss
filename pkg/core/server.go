// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import (
	"context"
	"net"
	"net/http"
)

// Server
type Server interface {
}

// ServerListener
type ServerListener interface {
	Name() string
	Listeners() []net.Listener
	RegisterListener(listener net.Listener) error
}

// ServerListenerModule
type ServerListenerModule interface {
	Module
	Register(listener ServerListener) error
	Serve(handler http.Handler) error
	Shutdown(ctx context.Context) error
	Close() error
}

// ServerSite
type ServerSite interface {
	Name() string
	Listeners() []string
	Hosts() []string
	Store() Store
	Fetcher() Fetcher
	Loader() Loader
	Server() Server
	RegisterMiddleware(middleware func(next http.Handler) http.Handler) error
	RegisterHandler(handler http.Handler) error
}

// ServerSiteHandlerModule
type ServerSiteHandlerModule interface {
	Module
	Register(server ServerSite) error
	Start() error
	Stop()
}

// ServerSiteMiddlewareModule
type ServerSiteMiddlewareModule interface {
	Module
	Register(server ServerSite) error
	Start() error
	Stop()
}
