package core

import "context"

// Loader
type Loader interface {
}

// LoaderParserModule
type LoaderParserModule interface {
	Module
	Parse(ctx context.Context, store Store, fetcher Fetcher) error
}
