package core

// Store is the interface of the store component.
//
// The store is responsible of the server state.
type Store interface {
	// Load a resource.
	LoadResource(name string) (*Resource, error)
	// Store a resource.
	StoreResource(name string, resource *Resource) error
}

// StoreStorageModule is the interface of a storage module.
type StoreStorageModule interface {
	// Module is the interface of a module.
	Module
	// Load a resource.
	LoadResource(name string) (*Resource, error)
	// Store a resource.
	StoreResource(name string, resource *Resource) error
}
