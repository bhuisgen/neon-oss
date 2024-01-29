// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// fetcher implements the fetcher.
type fetcher struct {
	config *fetcherConfig
	logger *slog.Logger
	state  *fetcherState
	mu     sync.RWMutex
}

// fetcherConfig implements the fetcher configuration.
type fetcherConfig struct {
	Providers map[string]map[string]map[string]interface{} `mapstructure:"providers"`
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
		logger: slog.New(NewLogHandler(os.Stderr, fetcherLogger, nil)),
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
			f.logger.Error("Failed to parse configuration", "err", err)
			return err
		}
	}

	var errInit bool

	for provider, providerConfig := range f.config.Providers {
		for moduleName, moduleConfig := range providerConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("fetcher.provider." + moduleName))
			if err != nil {
				f.logger.Error("Unregistered provider module", "provider", provider, "module", moduleName, "err", err)
				errInit = true
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.FetcherProviderModule)
			if !ok {
				err := errors.New("module instance not valid")
				f.logger.Error("Invalid provider module", "provide", provider, "module", moduleName, "err", err)
				errInit = true
				continue
			}

			if moduleConfig == nil {
				moduleConfig = map[string]interface{}{}
			}
			if err := module.Init(
				moduleConfig,
				slog.New(NewLogHandler(os.Stderr, fetcherLogger, nil)).With("provider", provider),
			); err != nil {
				f.logger.Error("Failed to init provider module", "provider", provider, "module", moduleName, "err", err)
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
	module, ok := f.state.providers[provider]
	f.mu.RUnlock()
	if !ok {
		err := errors.New("provider not found")
		return nil, err
	}

	resource, err := module.Fetch(ctx, name, config)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
