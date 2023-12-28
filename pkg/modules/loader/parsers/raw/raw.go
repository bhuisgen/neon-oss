// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package raw

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

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
	Resource map[string]map[string]interface{}
}

const (
	rawModuleID module.ModuleID = "loader.parser.raw"
	rawLogger   string          = "loader.parser.raw"
)

// init initializes the module.
func init() {
	module.Register(rawParser{})
}

// ModuleInfo returns the module information.
func (e rawParser) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: rawModuleID,
		NewInstance: func() module.Module {
			return &rawParser{}
		},
	}
}

// Check checks the parser configuration.
func (e *rawParser) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c rawParserConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.Resource == nil {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "Resource", c.Resource))
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the parser.
func (e *rawParser) Load(config map[string]interface{}) error {
	var c rawParserConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	e.config = &c
	e.logger = log.New(os.Stderr, fmt.Sprint(rawLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	return nil
}

// Parse parses a resource.
func (e *rawParser) Parse(ctx context.Context, store core.Store, fetcher core.Fetcher) error {
	var resourceName, resourceProvider string
	var resourceConfig map[string]interface{}
	for k := range e.config.Resource {
		resourceName = k
		break
	}
	if resourceName == "" {
		return errors.New("failed to parse resource name")
	}
	for k := range e.config.Resource[resourceName] {
		resourceProvider = k
		break
	}
	if resourceProvider == "" {
		return errors.New("failed to parse resource provider")
	}
	resourceConfig, _ = e.config.Resource[resourceName][resourceProvider].(map[string]interface{})

	resource, err := fetcher.Fetch(ctx, resourceName, resourceProvider, resourceConfig)
	if err != nil {
		return err
	}

	err = store.Set(resourceName, resource, resource.TTL)
	if err != nil {
		return err
	}

	return nil
}

var _ core.LoaderParserModule = (*rawParser)(nil)
