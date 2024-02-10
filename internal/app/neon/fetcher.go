package neon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// fetcher implements the fetcher.
type fetcher struct {
	config *fetcherConfig
	logger *slog.Logger
	state  *fetcherState
	mu     *sync.RWMutex
}

// fetcherConfig implements the fetcher configuration.
type fetcherConfig struct {
	Providers map[string]map[string]map[string]interface{} `mapstructure:"providers"`
}

// fetcherState implements the fetcher state.
type fetcherState struct {
	providers map[string]core.FetcherProviderModule
	mediator  *fetcherMediator
}

const (
	fetcherModuleID module.ModuleID = "app.fetcher"
)

// ModuleInfo returns the module information.
func (f fetcher) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: fetcherModuleID,
		NewInstance: func() module.Module {
			return &fetcher{
				logger: slog.New(log.NewHandler(os.Stderr, string(fetcherModuleID), nil)),
				state: &fetcherState{
					providers: make(map[string]core.FetcherProviderModule),
				},
				mu: &sync.RWMutex{},
			}
		},
	}
}

// Init initializes the fetcher.
func (f *fetcher) Init(config map[string]interface{}) error {
	f.logger.Debug("Initializing fetcher")

	if config == nil {
		f.config = &fetcherConfig{}
	} else {
		if err := mapstructure.Decode(config, &f.config); err != nil {
			f.logger.Error("Failed to parse configuration", "err", err)
			return fmt.Errorf("parse config: %w", err)
		}
	}

	var errConfig bool

	for provider, providerConfig := range f.config.Providers {
		for moduleName, moduleConfig := range providerConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("app.fetcher.provider." + moduleName))
			if err != nil {
				f.logger.Error("Unregistered provider module", "provider", provider, "module", moduleName, "err", err)
				errConfig = true
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.FetcherProviderModule)
			if !ok {
				err := errors.New("module instance not valid")
				f.logger.Error("Invalid provider module", "provide", provider, "module", moduleName, "err", err)
				errConfig = true
				continue
			}

			if moduleConfig == nil {
				moduleConfig = map[string]interface{}{}
			}
			if err := module.Init(moduleConfig); err != nil {
				f.logger.Error("Failed to init provider module", "provider", provider, "module", moduleName, "err", err)
				errConfig = true
				continue
			}

			f.state.providers[provider] = module

			break
		}
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the fetcher.
func (f *fetcher) Register(app core.App) error {
	f.state.mediator = newFetcherMediator(f)

	return nil
}

// Fetch fetches a resource from his name, provider and configuration.
func (f *fetcher) Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (
	*core.Resource, error) {
	f.logger.Debug("Fetching resource", "name", name, "provider", provider)

	f.mu.RLock()
	module, ok := f.state.providers[provider]
	f.mu.RUnlock()
	if !ok {
		return nil, errors.New("provider not found")
	}

	resource, err := module.Fetch(ctx, name, config)
	if err != nil {
		return nil, fmt.Errorf("fetch resource %s: %w", name, err)
	}

	return resource, nil
}

var _ Fetcher = (*fetcher)(nil)

// fetcherMediator implements the fetcher mediator.
type fetcherMediator struct {
	fetcher *fetcher
	mu      sync.RWMutex
}

// newFetcherMediator creates a new mediator.
func newFetcherMediator(fetcher *fetcher) *fetcherMediator {
	return &fetcherMediator{
		fetcher: fetcher,
	}
}

// Fetch fetches a resource from his name, provider and configuration.
func (m *fetcherMediator) Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (*core.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.fetcher.Fetch(ctx, name, provider, config)
}

var _ core.Fetcher = (*fetcherMediator)(nil)
