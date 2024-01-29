package core

import (
	"log/slog"

	"github.com/bhuisgen/neon/pkg/module"
)

// Module
type Module interface {
	module.Module
	Init(config map[string]interface{}, logger *slog.Logger) error
}
