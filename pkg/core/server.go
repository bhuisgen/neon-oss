package core

import (
	"context"
	"net"
	"net/http"
)

// Server is the interface of the server component.
//
// The server is composed of listeners and sites.
type Server interface {
}

// ServerListener is the interface of a listener.
//
// A listener is a network socket waiting for incoming connections.
type ServerListener interface {
	// Returns the listener name.
	Name() string
	// Returns the listener listeners.
	Listeners() []net.Listener
	// RegisterListener registers a listener.
	RegisterListener(listener net.Listener) error
}

// ServerListenerModule is the interface of a listener module.
type ServerListenerModule interface {
	// Module is the interface of a module.
	Module
	// Register registers the listener.
	Register(listener ServerListener) error
	// Serve accepts incoming connections.
	Serve(handler http.Handler) error
	// Shutdown shutdowns the listener gracefully.
	Shutdown(ctx context.Context) error
	// Close closes the listener.
	Close() error
}

// ServerSite is the interface a site.
type ServerSite interface {
	// Name returns the site name.
	Name() string
	// Listeners returns the site listeners.
	Listeners() []string
	// Hosts returns the site hosts.
	Hosts() []string
	// Store returns the store.
	Store() Store
	// Server returns the server.
	Server() Server
	// RegisterMiddleware registers a middleware.
	RegisterMiddleware(middleware func(next http.Handler) http.Handler) error
	// RegisterHandler registers a handler.
	RegisterHandler(handler http.Handler) error
}

// ServerSiteHandlerModule is the interface of a handler module.
type ServerSiteHandlerModule interface {
	// Module is the interface of a module.
	Module
	// Register registers the handler.
	Register(server ServerSite) error
	// Start starts the handler.
	Start() error
	// Stop stops the handler.
	Stop() error
}

// ServerSiteMiddlewareModule is the interface of a middleware module
type ServerSiteMiddlewareModule interface {
	// Module is the interface of a module.
	Module
	// Register registers the middleware.
	Register(server ServerSite) error
	// Start starts the middleware.
	Start() error
	// Stop stops the middleware.
	Stop() error
}
