package file

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

// fileProvider implements the file provider.
type fileProvider struct {
	config     *fileProviderConfig
	logger     *slog.Logger
	osReadFile func(name string) ([]byte, error)
}

// fileProviderConfig implements the file provider configuration.
type fileProviderConfig struct {
}

// fileResourceConfig implements the file resource configuration.
type fileResourceConfig struct {
	Path string `mapstructure:"path"`
}

const (
	fileModuleID module.ModuleID = "app.fetcher.provider.file"
)

// fileOsReadFile redirects to os.ReadFile.
func fileOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// init initializes the package.
func init() {
	module.Register(fileProvider{})
}

// ModuleInfo returns the module information.
func (p fileProvider) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           fileModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &fileProvider{
				logger:     slog.New(log.NewHandler(os.Stderr, string(fileModuleID), nil)),
				osReadFile: fileOsReadFile,
			}
		},
	}
}

// Init initializes the provider.
func (p *fileProvider) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &p.config); err != nil {
		p.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Fetch fetches a resource with the given configuration.
func (p *fileProvider) Fetch(ctx context.Context, name string, config map[string]interface{}) (*core.Resource, error) {
	var cfg fileResourceConfig
	if err := mapstructure.Decode(config, &cfg); err != nil {
		return nil, fmt.Errorf("parse resource %s config: %v", name, err)
	}

	data, err := p.osReadFile(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %v", cfg.Path, err)
	}

	return &core.Resource{
		Data: [][]byte{data},
		TTL:  0,
	}, nil
}

var _ core.FetcherProviderModule = (*fileProvider)(nil)
