// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/cache/memory"
	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/mitchellh/mapstructure"
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
	memoryLogger   string          = "store.storage.memory"
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

// Check checks the storage configuration.
func (s *memoryStorage) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c memoryStorageConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the provider.
func (s *memoryStorage) Load(config map[string]interface{}) error {
	var c memoryStorageConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	s.config = &c
	s.logger = log.New(os.Stderr, fmt.Sprint(memoryLogger, ": "), log.LstdFlags|log.Lmsgprefix)
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
