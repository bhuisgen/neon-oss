package memory

import (
	"errors"
	"log/slog"
	"os"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// memoryStorage implements the memory storage.
type memoryStorage struct {
	logger  *slog.Logger
	storage Cache
}

const (
	memoryModuleID module.ModuleID = "app.store.storage.memory"
)

// init initializes the package.
func init() {
	module.Register(memoryStorage{})
}

// ModuleInfo returns the module information.
func (s memoryStorage) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: memoryModuleID,
		NewInstance: func() module.Module {
			return &memoryStorage{
				logger: slog.New(log.NewHandler(os.Stderr, string(memoryModuleID), nil)),
			}
		},
	}
}

// Init initialize the storage.
func (s *memoryStorage) Init(config map[string]interface{}) error {
	s.storage = newCache()

	return nil
}

// LoadResource loads a resource from the storage.
func (s *memoryStorage) LoadResource(name string) (*core.Resource, error) {
	v := s.storage.Get(name)
	data, ok := v.(*core.Resource)
	if !ok {
		return nil, errors.New("no resource")
	}
	return data, nil
}

// StoreResource stores a resource into the storage.
func (s *memoryStorage) StoreResource(name string, resource *core.Resource) error {
	s.storage.Set(name, resource)
	return nil
}

var _ core.StoreStorageModule = (*memoryStorage)(nil)
