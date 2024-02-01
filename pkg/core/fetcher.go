package core

import "context"

// Fetcher is the interface of the fetcher component.
//
// The fetcher component fetches resources through providers.
type Fetcher interface {
	// Fetch fetches a resource from his name, provider and configuration.
	Fetch(ctx context.Context, name string, provider string,
		config map[string]interface{}) (*Resource, error)
}

// FetcherProviderModule is the interface of a provider module.
type FetcherProviderModule interface {
	// Module is the interface of a module.
	Module

	// Fetch fetches a resource with the given configuration.
	Fetch(ctx context.Context, name string, config map[string]interface{}) (
		*Resource, error)
}
