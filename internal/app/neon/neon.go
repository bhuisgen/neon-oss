package neon

import (
	"log/slog"
	"os"

	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

var (
	DEBUG bool = false

	CHILD_SOCKET string = "neon.sock"
)

// init initializes the package.
func init() {
	module.Register(app{})
}

// New creates a new instance.
func New(config *config) App {
	if _, ok := os.LookupEnv("DEBUG"); ok {
		DEBUG = true
	}
	if v, ok := os.LookupEnv("CHILD_SOCKET"); ok {
		CHILD_SOCKET = v
	}

	if DEBUG {
		log.ProgramLevel.Set(slog.LevelDebug)
	}

	appModuleInfo, err := module.Lookup("app")
	if err != nil {
		log.Fatalf("Failed to lookup app module: %v", err)
	}
	app, ok := appModuleInfo.NewInstance().(App)
	if !ok {
		log.Fatal("Failed to create app instance")
	}
	cfg, ok := config.data["app"].(map[string]interface{})
	if !ok {
		log.Fatal("Missing app configuration")
	}
	if err := app.Init(cfg); err != nil {
		log.Fatalf("Failed to init app: %v", err)
	}

	return app
}
