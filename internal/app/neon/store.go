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
	Storage map[string]map[string]interface{}
}

// storeState implements the datastore state
type storeState struct {
	storage       string
	storageModule core.StoreStorageModule
}

const (
	storeLogger string = "store"
)

// newStore creates a new store.
func newStore() *store {
	return &store{}
}

// Check checks the store configuration.
func (s *store) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c storeConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "store: failed to parse configuration")
		return report, err
	}

	if len(c.Storage) == 0 {
		report = append(report, "store: no storage defined")
	}

	for storage, storageConfig := range c.Storage {
		moduleInfo, err := module.Lookup(module.ModuleID("store.storage." + storage))
		if err != nil {
			report = append(report, fmt.Sprintf("store: unregistered storage module '%s'", storage))
			continue
		}
		module, ok := moduleInfo.NewInstance().(core.StoreStorageModule)
		if !ok {
			report = append(report, fmt.Sprintf("store: invalid storage module '%s'", storage))
			continue
		}
		r, err := module.Check(storageConfig)
		if err != nil {
			for _, line := range r {
				report = append(report, fmt.Sprintf("store: failed to check configuration: %s", line))
			}
			continue
		}

		break
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}
	return nil, nil
}

// Load loads the store.
func (s *store) Load(config map[string]interface{}) error {
	var c storeConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	s.config = &c
	s.logger = log.New(os.Stderr, fmt.Sprint(storeLogger, ": "), log.LstdFlags|log.Lmsgprefix)
	s.state = &storeState{}

	for storage, storageConfig := range s.config.Storage {
		moduleInfo, err := module.Lookup(module.ModuleID("store.storage." + storage))
		if err != nil {
			return err
		}
		module, ok := moduleInfo.NewInstance().(core.StoreStorageModule)
		if !ok {
			return fmt.Errorf("invalid storage module '%s'", storage)
		}
		err = module.Load(storageConfig)
		if err != nil {
			return err
		}

		s.state.storage = storage
		s.state.storageModule = module

		break
	}

	return nil
}

// LoadResource loads a resource.
func (s *store) LoadResource(name string) (*core.Resource, error) {
	resource, err := s.state.storageModule.LoadResource(name)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

// StoreResource stores a resource.
func (s *store) StoreResource(name string, resource *core.Resource) error {
	err := s.state.storageModule.StoreResource(name, resource)
	if err != nil {
		return err
	}

	return nil
}

var _ Store = (*store)(nil)
