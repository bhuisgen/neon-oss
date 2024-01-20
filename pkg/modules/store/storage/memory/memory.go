// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"errors"
	"log"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/cache/memory"
	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// memoryStorage implements the memory storage.
type memoryStorage struct {
	config  *memoryStorageConfig
	logger  *log.Logger
	storage cache.Cache
}

// memoryStorageConfig implements the memory storage configuration.
type memoryStorageConfig struct {
}

const (
	memoryModuleID module.ModuleID = "store.storage.memory"
)

// init initializes the module.
func init() {
	module.Register(memoryStorage{})
}

// ModuleInfo returns the module information.
func (s memoryStorage) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: memoryModuleID,
		NewInstance: func() module.Module {
			return &memoryStorage{}
		},
	}
}

// Init initialize the storage.
func (s *memoryStorage) Init(config map[string]interface{}, logger *log.Logger) error {
	s.logger = logger

	if err := mapstructure.Decode(config, &s.config); err != nil {
		s.logger.Print("failed to parse configuration")
	}

	s.storage = memory.New(0, 0)

	return nil
}

// LoadResource loads a resource from the storage.
func (s *memoryStorage) LoadResource(name string) (*core.Resource, error) {
	v := s.storage.Get(name)
	data, ok := v.(*core.Resource)
	if !ok {
		return nil, errors.New("invalid data")
	}
	return data, nil
}

// StoreResource stores a resource into the storage.
func (s *memoryStorage) StoreResource(name string, resource *core.Resource) error {
	s.storage.Set(name, resource)
	return nil
}

var _ core.StoreStorageModule = (*memoryStorage)(nil)
