// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// jsonParser implements the json parser.
type jsonParser struct {
	config        *jsonParserConfig
	logger        *log.Logger
	jsonUnmarshal func(data []byte, v any) error
}

// jsonParserConfig implements the json parser configuration.
type jsonParserConfig struct {
	Resource     map[string]map[string]interface{}
	Filter       string
	ItemParams   map[string]string
	ItemResource map[string]map[string]interface{}
}

const (
	jsonModuleID module.ModuleID = "loader.parser.json"
	jsonLogger   string          = "loader.parser.json"
)

// loaderJsonUnmarshal redirects to json.Unmarshal.
func loaderJsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// init initializes the module.
func init() {
	module.Register(jsonParser{})
}

// ModuleInfo implements module.Module.
func (e jsonParser) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: jsonModuleID,
		NewInstance: func() module.Module {
			return &jsonParser{
				jsonUnmarshal: loaderJsonUnmarshal,
			}
		},
	}
}

// Check implements core.Module.
func (e *jsonParser) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c jsonParserConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.Resource == nil {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "Resource", c.Resource))
	}
	if c.Filter == "" {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "Filter", c.Filter))
	}
	if c.ItemResource == nil {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "ItemResource", c.ItemResource))
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the parser.
func (e *jsonParser) Load(config map[string]interface{}) error {
	var c jsonParserConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	e.config = &c
	e.logger = log.New(os.Stderr, fmt.Sprint(jsonLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	return nil
}

// Parse parses a resource.
func (e *jsonParser) Parse(ctx context.Context, store core.Store, fetcher core.Fetcher) error {
	var resourceName, resourceProvider string
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

	var resourceConfig map[string]interface{}
	resourceConfig, _ = e.config.Resource[resourceName][resourceProvider].(map[string]interface{})
	resource, err := fetcher.Fetch(ctx, resourceName, resourceProvider, resourceConfig)
	if err != nil {
		return err
	}

	for _, data := range resource.Data {
		var jsonData interface{}
		err = e.jsonUnmarshal(data, &jsonData)
		if err != nil {
			return err
		}
		result, err := jsonpath.Get(e.config.Filter, jsonData)
		if err != nil {
			return err
		}
		switch v := result.(type) {
		case []interface{}:
			for _, item := range v {
				mItem, ok := item.(map[string]interface{})
				if !ok {
					return errors.New("failed to parse item")
				}
				err := e.executeResourceFromItem(ctx, store, fetcher, mItem)
				if err != nil {
					return err
				}
			}
		}
	}

	err = store.Set(resourceName, resource, resource.TTL)
	if err != nil {
		return err
	}

	return nil
}

// executeResourceFromItem loads a resource from the given item.
func (e *jsonParser) executeResourceFromItem(ctx context.Context, store core.Store, fetcher core.Fetcher,
	item map[string]interface{}) error {
	var params map[string]interface{}
	for k, v := range e.config.ItemParams {
		data, err := jsonpath.Get(v, item)
		if err != nil || data == nil {
			continue
		}
		if params == nil {
			params = make(map[string]interface{})
		}
		params[k] = data
	}

	var itemResourceName, itemResourceProvider string
	var itemResourceConfig map[string]interface{}
	for k := range e.config.ItemResource {
		itemResourceName = k
		break
	}
	if itemResourceName == "" {
		return errors.New("failed to parse item resource name")
	}
	for k := range e.config.ItemResource[itemResourceName] {
		itemResourceProvider = k
		break
	}
	if itemResourceProvider == "" {
		return errors.New("failed to parse item resource provider")
	}
	itemResourceConfig, _ = e.config.ItemResource[itemResourceName][itemResourceProvider].(map[string]interface{})

	itemResourceName = replaceParameters(itemResourceName, params)
	itemResourceProvider = replaceParameters(itemResourceProvider, params)
	itemResourceConfig = replaceParametersInMap(itemResourceConfig, params)

	resource, err := fetcher.Fetch(ctx, itemResourceName, itemResourceProvider, itemResourceConfig)
	if err != nil {
		return err
	}

	err = store.Set(itemResourceName, resource, resource.TTL)
	if err != nil {
		return err
	}

	return nil
}

var _ core.LoaderParserModule = (*jsonParser)(nil)

// replaceParameters returns a copy of the string s with all its parameters replaced.
func replaceParameters(s string, params map[string]interface{}) string {
	t := s
	for k, v := range params {
		var value string
		switch vt := v.(type) {
		case string:
			value = vt
		case int:
			value = strconv.FormatInt(int64(vt), 10)
		case float64:
			value = strconv.FormatFloat(vt, 'f', -1, 64)
		case bool:
			value = strconv.FormatBool(vt)
		}
		t = strings.ReplaceAll(t, fmt.Sprint("$", k), value)
	}
	return t
}

// replaceParametersInMap returns a copy of the map m with all its parameters replaced.
func replaceParametersInMap(m map[string]interface{}, params map[string]interface{}) map[string]interface{} {
	t := make(map[string]interface{}, len(m))
	for k, v := range m {
		key := replaceParameters(k, params)
		switch vt := v.(type) {
		case []map[string]interface{}:
			arr := make([]map[string]interface{}, len(vt))
			for index, m := range vt {
				arr[index] = replaceParametersInMap(m, params)
			}
			t[key] = arr
		case map[string]interface{}:
			t[key] = replaceParametersInMap(vt, params)
		case string:
			value := replaceParameters(vt, params)
			t[key] = value
		default:
			t[key] = vt
		}
	}
	return t
}
