package neon

import (
	"context"
	"net/http"
	"os"

	"github.com/bhuisgen/neon/pkg/core"
)

// Application
type Application interface {
	Check() error
	Serve() error
}

// Store
type Store interface {
	Init(config map[string]interface{}) error
	LoadResource(name string) (*core.Resource, error)
	StoreResource(name string, resource *core.Resource) error
}

// Fetcher
type Fetcher interface {
	Init(config map[string]interface{}) error
	Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (*core.Resource, error)
}

// Loader
type Loader interface {
	Init(config map[string]interface{}) error
	Start() error
	Stop() error
}

// Server
type Server interface {
	Init(config map[string]interface{}) error
	Register(descriptors map[string]ServerListenerDescriptor) error
	Start() error
	Stop() error
	Shutdown(ctx context.Context) error
}

// ServerListener
type ServerListener interface {
	Init(config map[string]interface{}) error
	Register(descriptor ServerListenerDescriptor) error
	Serve() error
	Shutdown(ctx context.Context) error
	Close() error
	Remove() error
	Name() string
	Link(site ServerSite) error
	Unlink(site ServerSite) error
	Descriptor() (ServerListenerDescriptor, error)
}

// ServerListenerRouter
type ServerListenerRouter interface {
	http.Handler
}

// ServerListenerDescriptor
type ServerListenerDescriptor interface {
	Files() []*os.File
}

// ServerSite
type ServerSite interface {
	Init(config map[string]interface{}) error
	Register() error
	Start() error
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
