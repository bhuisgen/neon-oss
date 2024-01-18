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
	providers        map[string]string
	providersModules map[string]core.FetcherProviderModule
}

const (
	fetcherLogger string = "fetcher"
)

// newFetcher creates a new fetcher.
func newFetcher() *fetcher {
	return &fetcher{}
}

// Check checks the fetcher configuration.
func (f *fetcher) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c fetcherConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "fetcher: failed to parse configuration")
		return report, err
	}

	for provider, providerConfig := range c.Providers {
		for moduleName, moduleConfig := range providerConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("fetcher.provider." + moduleName))
			if err != nil {
				report = append(report, fmt.Sprintf("fetcher: provider '%s', unregistered provider module '%s'", provider,
					moduleName))
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.FetcherProviderModule)
			if !ok {
				report = append(report, fmt.Sprintf("fetcher: provider '%s', invalid provider module '%s'", provider,
					moduleName))
				continue
			}
			r, err := module.Check(moduleConfig)
			if err != nil {
				for _, line := range r {
					report = append(report, fmt.Sprintf("fetcher: provider '%s', failed to check configuration: %s", provider,
						line))
				}
				continue
			}

			break
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the fetcher.
func (f *fetcher) Load(config map[string]interface{}) error {
	var c fetcherConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	f.config = &c
	f.logger = log.New(os.Stderr, fmt.Sprint(fetcherLogger, ": "), log.LstdFlags|log.Lmsgprefix)
	f.state = &fetcherState{
		providers:        make(map[string]string),
		providersModules: make(map[string]core.FetcherProviderModule),
	}

	for provider, providerConfig := range c.Providers {
		for moduleName, moduleConfig := range providerConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("fetcher.provider." + moduleName))
			if err != nil {
				return err
			}
			module, ok := moduleInfo.NewInstance().(core.FetcherProviderModule)
			if !ok {
				return fmt.Errorf("provider '%s', invalid provider module '%s'", provider, moduleName)
			}
			err = module.Load(moduleConfig)
			if err != nil {
				return err
			}

			f.state.providers[provider] = moduleName
			f.state.providersModules[moduleName] = module

			break
		}
	}

	return nil
}

// Fetch fetches a registered resource.
func (f *fetcher) Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (*core.Resource, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	providerModule, ok := f.state.providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider not found")
	}
	module, ok := f.state.providersModules[providerModule]
	if !ok {
		return nil, fmt.Errorf("provider module not found")
	}

	resource, err := module.Fetch(ctx, name, config)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
