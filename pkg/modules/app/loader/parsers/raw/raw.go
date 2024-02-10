package raw

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// rawParser implements the raw parser.
type rawParser struct {
	config *rawParserConfig
	logger *slog.Logger
}

// rawParserConfig implements the raw parser configuration.
type rawParserConfig struct {
	Resource map[string]map[string]interface{} `mapstructure:"resource"`
}

const (
	rawModuleID module.ModuleID = "app.loader.parser.raw"
)

// init initializes the package.
func init() {
	module.Register(rawParser{})
}

// ModuleInfo returns the module information.
func (p rawParser) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: rawModuleID,
		NewInstance: func() module.Module {
			return &rawParser{
				logger: slog.New(log.NewHandler(os.Stderr, string(rawModuleID), nil)),
			}
		},
	}
}

// Init initializes the parser configuration.
func (p *rawParser) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &p.config); err != nil {
		p.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if p.config.Resource == nil {
		p.logger.Error("Invalid value", "option", "Resource", "value", p.config.Resource)
		errConfig = true
	}

	if errConfig {
		return errors.New("config")
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
		return errors.New("invalid resource name")
	}
	for k := range p.config.Resource[resourceName] {
		resourceProvider = k
		break
	}
	if resourceProvider == "" {
		return errors.New("invalid resource provider")
	}
	resourceConfig, _ = p.config.Resource[resourceName][resourceProvider].(map[string]interface{})

	resource, err := fetcher.Fetch(ctx, resourceName, resourceProvider, resourceConfig)
	if err != nil {
		return fmt.Errorf("fetch resource %s: %v", resourceName, err)
	}

	if err := store.StoreResource(resourceName, resource); err != nil {
		return fmt.Errorf("store resource %s: %v", resourceName, err)
	}

	return nil
}

var _ core.LoaderParserModule = (*rawParser)(nil)
