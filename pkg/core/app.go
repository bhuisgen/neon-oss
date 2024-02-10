package core

import (
	"net"
)

// App is the interface of the app component.
//
// The app component is the main component who loads all base components:
// store, fetcher, loader and server.
type App interface {
	// Returns the store.
	Store() Store
	// Returns the fetcher.
	Fetcher() Fetcher
	// Returns the loader.
	Loader() Loader
	// Returns the server.
	Server() Server
	// Returns the network listeners.
	Listeners() map[string][]net.Listener
}

// AppModule is the interface of an app module.
type AppModule interface {
	// Module is the interface of a module.
	Module
	// Register registers the module.
	Register(app App) error
}
