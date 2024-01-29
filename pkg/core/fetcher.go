package core

import "context"

// Fetcher
type Fetcher interface {
	Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (*Resource, error)
}

// FetcherProviderModule
type FetcherProviderModule interface {
	Module
	Fetch(ctx context.Context, name string, config map[string]interface{}) (*Resource, error)
}
