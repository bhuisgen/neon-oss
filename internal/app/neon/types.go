package neon

import (
	"context"
	"net"
	"net/http"

	"github.com/bhuisgen/neon/pkg/core"
)

// App
type App interface {
	core.Module
	Check() error
	Serve(ctx context.Context) error
}

// Store
type Store interface {
	core.AppModule
	LoadResource(name string) (*core.Resource, error)
	StoreResource(name string, resource *core.Resource) error
}

// Fetcher
type Fetcher interface {
	core.AppModule
	Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (*core.Resource, error)
}

// Loader
type Loader interface {
	core.AppModule
	Start(ctx context.Context) error
	Stop() error
}

// Server
type Server interface {
	core.AppModule
	Start(ctx context.Context) error
	Stop() error
	Shutdown(ctx context.Context) error
	Listeners() (map[string][]net.Listener, error)
}

// ServerListener
type ServerListener interface {
	Init(config map[string]interface{}) error
	Register(core.App) error
	Serve() error
	Shutdown(ctx context.Context) error
	Close() error
	Remove() error
	Name() string
	Link(site ServerSite) error
	Unlink(site ServerSite) error
	Listeners() ([]net.Listener, error)
}

// ServerListenerRouter
type ServerListenerRouter interface {
	http.Handler
}

// ServerSite
type ServerSite interface {
	Init(config map[string]interface{}) error
	Register(core.App) error
	Start(ctx context.Context) error
	Stop() error
	Name() string
	Listeners() []string
	Hosts() []string
	Router() (ServerSiteRouter, error)
}

// ServerSiteRouter
type ServerSiteRouter interface {
	Routes() map[string]http.Handler
}
