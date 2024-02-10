package core

import (
	"github.com/bhuisgen/neon/pkg/module"
)

// Module is the interface of a module.
type Module interface {
	// Module is the base interface of a module.
	module.Module
	// Init initializes a module with the given configuration.
	Init(config map[string]interface{}) error
}
