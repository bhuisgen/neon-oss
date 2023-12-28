// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/core"
)

// store implements the datastore
type store struct {
	config *storeConfig
	logger *log.Logger
	state  *storeState
}

// storeConfig implements the datastore configuration
type storeConfig struct {
}

// storeState implements the datastore state
type storeState struct {
	data cache.Cache
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
	s.state = &storeState{
		data: cache.NewCache(),
	}

	return nil
}

// Get returns the data.
func (s *store) Get(name string) (*core.Resource, error) {
	v := s.state.data.Get(name)
	if v == nil {
		return nil, errors.New("no data")
	}

	r, ok := v.(*core.Resource)
	if !ok {
		return nil, errors.New("invalid data")
	}

	return r, nil
}

// Set stores the data.
func (s *store) Set(name string, resource *core.Resource, ttl time.Duration) error {
	s.state.data.Set(name, resource, ttl)

	return nil
}

var _ Store = (*store)(nil)
