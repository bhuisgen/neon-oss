// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// store implements the datastore
type store struct {
	config *storeConfig
	logger *log.Logger
	state  *storeState
}

// storeConfig implements the datastore configuration
type storeConfig struct {
	Storage map[string]map[string]interface{} `mapstructure:"storage"`
}

// storeState implements the datastore state
type storeState struct {
	storage core.StoreStorageModule
}

const (
	storeLogger string = "store"
)

// newStore creates a new store.
func newStore() *store {
	return &store{
		logger: log.New(os.Stderr, fmt.Sprint(storeLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		state:  &storeState{},
	}
}

// Init initializes the store.
func (s *store) Init(config map[string]interface{}) error {
	if config == nil {
		s.config = &storeConfig{
			Storage: map[string]map[string]interface{}{
				"memory": {},
			},
		}
	} else {
		if err := mapstructure.Decode(config, &s.config); err != nil {
			s.logger.Print("failed to parse configuration")
			return err
		}
	}

	var errInit bool

	if len(s.config.Storage) == 0 {
		s.logger.Print("no storage defined")
		errInit = true
	}
	for storage, storageConfig := range s.config.Storage {
		moduleInfo, err := module.Lookup(module.ModuleID("store.storage." + storage))
		if err != nil {
			s.logger.Printf("unregistered storage module '%s'", storage)
			errInit = true
			break
		}
		module, ok := moduleInfo.NewInstance().(core.StoreStorageModule)
		if !ok {
			s.logger.Printf("invalid storage module '%s'", storage)
			errInit = true
			break
		}

		if storageConfig == nil {
			storageConfig = map[string]interface{}{}
		}
		if err := module.Init(
			storageConfig,
			log.New(os.Stderr, fmt.Sprint(s.logger.Prefix(), "storage[", storage, "]: "), log.LstdFlags|log.Lmsgprefix),
		); err != nil {
			s.logger.Printf("failed to init storage module '%s'", storage)
			errInit = true
			break
		}

		s.state.storage = module

		break
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// LoadResource loads a resource.
func (s *store) LoadResource(name string) (*core.Resource, error) {
	resource, err := s.state.storage.LoadResource(name)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

// StoreResource stores a resource.
func (s *store) StoreResource(name string, resource *core.Resource) error {
	if err := s.state.storage.StoreResource(name, resource); err != nil {
		return err
	}

	return nil
}

var _ Store = (*store)(nil)
