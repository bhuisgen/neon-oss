package core

import (
	"context"
)

// Loader is the interface of the loader component.
//
// The loader executes all the rules to fetch and to store resources.
//
// Each loader rule is processed by a parser module which triggers the fetcher
// component to fetch resources and next the state component to store the
// resources into the server state.
type Loader interface {
}

// LoaderParserModule
type LoaderParserModule interface {
	// Module is the interface of a module.
	Module
	// Parse parses a resource.
	Parse(ctx context.Context, store Store, fetcher Fetcher) error
}
