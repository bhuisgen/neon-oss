package neon

import (
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

// store implements the store.
type store struct {
	config *storeConfig
	logger *slog.Logger
	state  *storeState
}

// storeConfig implements the store configuration.
type storeConfig struct {
	Storage map[string]map[string]interface{} `mapstructure:"storage"`
}

// storeState implements the store state.
type storeState struct {
	storage  core.StoreStorageModule
	mediator *storeMediator
}

const (
	storeModuleID module.ModuleID = "app.store"
)

// ModuleInfo returns the module information.
func (s store) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           storeModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &store{
				logger: slog.New(log.NewHandler(os.Stderr, string(storeModuleID), nil)),
				state:  &storeState{},
			}
		},
	}
}

// Init initializes the store.
func (s *store) Init(config map[string]interface{}) error {
	s.logger.Debug("Initializing store")

	if config == nil {
		s.config = &storeConfig{
			Storage: map[string]map[string]interface{}{
				"memory": {},
			},
		}
	} else {
		if err := mapstructure.Decode(config, &s.config); err != nil {
			s.logger.Error("Failed to parse configuration", "err", err)
			return fmt.Errorf("parse config: %w", err)
		}
	}

	var errConfig bool

	if len(s.config.Storage) == 0 {
		s.logger.Error("No storage defined")
		errConfig = true
	}
	for storage, storageConfig := range s.config.Storage {
		moduleInfo, err := module.Lookup(module.ModuleID("app.store.storage." + storage))
		if err != nil {
			s.logger.Error("Unregistered storage module", "module", storage, "err", err)
			errConfig = true
			break
		}
		module, ok := moduleInfo.NewInstance().(core.StoreStorageModule)
		if !ok {
			err := errors.New("module instance not valid")
			s.logger.Error("Invalid storage module", "module", storage, "err", err)
			errConfig = true
			break
		}

		if storageConfig == nil {
			storageConfig = map[string]interface{}{}
		}
		if err := module.Init(storageConfig); err != nil {
			s.logger.Error("Failed to init storage module", "module", storage, "err", err)
			errConfig = true
			break
		}

		s.state.storage = module

		break
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the store.
func (s *store) Register(app core.App) error {
	s.state.mediator = newStoreMediator(s)

	return nil
}

// LoadResource loads a resource.
func (s *store) LoadResource(name string) (*core.Resource, error) {
	s.logger.Debug("Loading resource", "name", name)

	resource, err := s.state.storage.LoadResource(name)
	if err != nil {
		return nil, fmt.Errorf("load resource: %w", err)
	}

	return resource, nil
}

// StoreResource stores a resource.
func (s *store) StoreResource(name string, resource *core.Resource) error {
	s.logger.Debug("Storing resource", "name", name)

	if err := s.state.storage.StoreResource(name, resource); err != nil {
		return fmt.Errorf("store resource: %w", err)
	}

	return nil
}

var _ Store = (*store)(nil)

// storeMediator implements the store mediator.
type storeMediator struct {
	store *store
	mu    sync.RWMutex
}

// newStoreMediator creates a new mediator.
func newStoreMediator(store *store) *storeMediator {
	return &storeMediator{
		store: store,
	}
}

// LoadResource loads a resource.
func (m *storeMediator) LoadResource(name string) (*core.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.store.LoadResource(name)
}

// StoreResource stores a resource.
func (m *storeMediator) StoreResource(name string, resource *core.Resource) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.store.StoreResource(name, resource)
}

var _ core.Store = (*storeMediator)(nil)
