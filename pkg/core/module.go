package core

import (
	"log/slog"

	"github.com/bhuisgen/neon/pkg/module"
)

// Module is the interface of a module.
type Module interface {
	// Module is the base interface of a module.
	module.Module

	// Init initializes a module with the given configuration and logger.
	Init(config map[string]interface{}, logger *slog.Logger) error
}
