package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	logger        *slog.Logger
	jsonUnmarshal func(data []byte, v any) error
}

// jsonParserConfig implements the json parser configuration.
type jsonParserConfig struct {
	Resource     map[string]map[string]interface{} `mapstructure:"resource"`
	Filter       string                            `mapstructure:"filter"`
	ItemParams   map[string]string                 `mapstructure:"itemParams"`
	ItemResource map[string]map[string]interface{} `mapstructure:"itemResource"`
}

const (
	jsonModuleID module.ModuleID = "loader.parser.json"
)

// loaderJsonUnmarshal redirects to json.Unmarshal.
func loaderJsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// init initializes the module.
func init() {
	module.Register(jsonParser{})
}

// ModuleInfo returns the module information.
func (p jsonParser) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: jsonModuleID,
		NewInstance: func() module.Module {
			return &jsonParser{
				jsonUnmarshal: loaderJsonUnmarshal,
			}
		},
	}
}

// Init initializes the parser.
func (p *jsonParser) Init(config map[string]interface{}, logger *slog.Logger) error {
	p.logger = logger

	if err := mapstructure.Decode(config, &p.config); err != nil {
		p.logger.Error("Failed to parse configuration")
		return err
	}

	var errInit bool

	if p.config.Resource == nil {
		p.logger.Error("Invalid value", "option", "Resource", "value", p.config.Resource)
		errInit = true
	}
	if p.config.Filter == "" {
		p.logger.Error("Invalid value", "option", "Filter", "value", p.config.Filter)
		errInit = true
	}
	if p.config.ItemResource == nil {
		p.logger.Error("Invalid value", "option", "ItemResource", "value", p.config.ItemResource)
		errInit = true
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Parse parses a resource.
func (p *jsonParser) Parse(ctx context.Context, store core.Store, fetcher core.Fetcher) error {
	var resourceName, resourceProvider string
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

	var resourceConfig map[string]interface{}
	resourceConfig, _ = p.config.Resource[resourceName][resourceProvider].(map[string]interface{})
	resource, err := fetcher.Fetch(ctx, resourceName, resourceProvider, resourceConfig)
	if err != nil {
		return err
	}

	for _, data := range resource.Data {
		var jsonData interface{}
		err = p.jsonUnmarshal(data, &jsonData)
		if err != nil {
			return err
		}
		result, err := jsonpath.Get(p.config.Filter, jsonData)
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
				err := p.executeResourceFromItem(ctx, store, fetcher, mItem)
				if err != nil {
					return err
				}
			}
		}
	}

	err = store.StoreResource(resourceName, resource)
	if err != nil {
		return err
	}

	return nil
}

// executeResourceFromItem loads a resource from the given item.
func (p *jsonParser) executeResourceFromItem(ctx context.Context, store core.Store, fetcher core.Fetcher,
	item map[string]interface{}) error {
	var params map[string]interface{}
	for k, v := range p.config.ItemParams {
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
	for k := range p.config.ItemResource {
		itemResourceName = k
		break
	}
	if itemResourceName == "" {
		return errors.New("failed to parse item resource name")
	}
	for k := range p.config.ItemResource[itemResourceName] {
		itemResourceProvider = k
		break
	}
	if itemResourceProvider == "" {
		return errors.New("failed to parse item resource provider")
	}
	itemResourceConfig, _ = p.config.ItemResource[itemResourceName][itemResourceProvider].(map[string]interface{})

	itemResourceName = replaceParameters(itemResourceName, params)
	itemResourceProvider = replaceParameters(itemResourceProvider, params)
	itemResourceConfig = replaceParametersInMap(itemResourceConfig, params)

	resource, err := fetcher.Fetch(ctx, itemResourceName, itemResourceProvider, itemResourceConfig)
	if err != nil {
		return err
	}

	err = store.StoreResource(itemResourceName, resource)
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
