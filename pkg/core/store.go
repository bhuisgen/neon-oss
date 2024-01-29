package core

// Store
type Store interface {
	LoadResource(name string) (*Resource, error)
	StoreResource(name string, resource *Resource) error
}

// StoreStorageModule
type StoreStorageModule interface {
	Module
	LoadResource(name string) (*Resource, error)
	StoreResource(name string, resource *Resource) error
}
