// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// fetcher implements the fetcher.
type fetcher struct {
	config *fetcherConfig
	logger *log.Logger
	state  *fetcherState
	mu     sync.RWMutex
}

// fetcherConfig implements the fetcher configuration.
type fetcherConfig struct {
	Providers map[string]map[string]map[string]interface{}
}

// fetcherState implements the fetcher state.
type fetcherState struct {
	providers map[string]core.FetcherProviderModule
}

const (
	fetcherLogger string = "fetcher"
)

// newFetcher creates a new fetcher.
func newFetcher() *fetcher {
	return &fetcher{
		logger: log.New(os.Stderr, fmt.Sprint(fetcherLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		state: &fetcherState{
			providers: make(map[string]core.FetcherProviderModule),
		},
	}
}

// Init initializes the fetcher.
func (f *fetcher) Init(config map[string]interface{}) error {
	if config == nil {
		f.config = &fetcherConfig{}
	} else {
		if err := mapstructure.Decode(config, &f.config); err != nil {
			f.logger.Print("failed to parse configuration")
			return err
		}
	}

	var errInit bool

	for provider, providerConfig := range f.config.Providers {
		for moduleName, moduleConfig := range providerConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("fetcher.provider." + moduleName))
			if err != nil {
				f.logger.Printf("provider '%s', unregistered module '%s'", provider, moduleName)
				errInit = true
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.FetcherProviderModule)
			if !ok {
				f.logger.Printf("provider '%s', invalid module '%s'", provider, moduleName)
				errInit = true
				continue
			}

			if moduleConfig == nil {
				moduleConfig = map[string]interface{}{}
			}
			if err := module.Init(
				moduleConfig,
				log.New(os.Stderr, fmt.Sprint(f.logger.Prefix(), "provider[", provider, "]: "), log.LstdFlags|log.Lmsgprefix),
			); err != nil {
				f.logger.Printf("provider '%s', failed to init module '%s'", provider, moduleName)
				errInit = true
				continue
			}

			f.state.providers[provider] = module

			break
		}
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Fetch fetches a resource.
func (f *fetcher) Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (
	*core.Resource, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	module, ok := f.state.providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider module not found")
	}

	resource, err := module.Fetch(ctx, name, config)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
