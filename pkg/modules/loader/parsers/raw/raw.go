// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package raw

import (
	"context"
	"errors"
	"log"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// rawParser implements the raw parser.
type rawParser struct {
	config *rawParserConfig
	logger *log.Logger
}

// rawParserConfig implements the raw parser configuration.
type rawParserConfig struct {
	Resource map[string]map[string]interface{} `mapstructure:"resource"`
}

const (
	rawModuleID module.ModuleID = "loader.parser.raw"
)

// init initializes the module.
func init() {
	module.Register(rawParser{})
}

// ModuleInfo returns the module information.
func (p rawParser) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: rawModuleID,
		NewInstance: func() module.Module {
			return &rawParser{}
		},
	}
}

// Init initializes the parser configuration.
func (p *rawParser) Init(config map[string]interface{}, logger *log.Logger) error {
	p.logger = logger

	if err := mapstructure.Decode(config, &p.config); err != nil {
		p.logger.Print("failed to parse configuration")
		return err
	}

	var errInit bool

	if p.config.Resource == nil {
		p.logger.Printf("option '%s', invalid value '%s'", "Resource", p.config.Resource)
		errInit = true
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Parse parses a resource.
func (p *rawParser) Parse(ctx context.Context, store core.Store, fetcher core.Fetcher) error {
	var resourceName, resourceProvider string
	var resourceConfig map[string]interface{}
	for k := range p.config.Resource {
		resourceName = k
		break
	}
	if resourceName == "" {
		return errors.New("failed to parse resource name")
	}
	for k := range p.config.Resource[resourceName] {
		resourceProvider = k
		break
	}
	if resourceProvider == "" {
		return errors.New("failed to parse resource provider")
	}
	resourceConfig, _ = p.config.Resource[resourceName][resourceProvider].(map[string]interface{})

	resource, err := fetcher.Fetch(ctx, resourceName, resourceProvider, resourceConfig)
	if err != nil {
		return err
	}

	err = store.StoreResource(resourceName, resource)
	if err != nil {
		return err
	}

	return nil
}

var _ core.LoaderParserModule = (*rawParser)(nil)
