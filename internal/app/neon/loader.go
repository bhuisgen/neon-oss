package neon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// loader implements the loader.
type loader struct {
	config *loaderConfig
	logger *slog.Logger
	state  *loaderState
	mu     *sync.RWMutex
	stop   chan struct{}
}

// loaderConfig implements the loader configuration.
type loaderConfig struct {
	ExecStartup          *int                                         `mapstructure:"execStartup"`
	ExecInterval         *int                                         `mapstructure:"execInterval"`
	ExecFailsafeInterval *int                                         `mapstructure:"execFailsafeInterval"`
	ExecWorkers          *int                                         `mapstructure:"execWorkers"`
	ExecMaxOps           *int                                         `mapstructure:"execMaxOps"`
	ExecMaxDelay         *int                                         `mapstructure:"execMaxDelay"`
	Rules                map[string]map[string]map[string]interface{} `mapstructure:"rules"`
}

// loaderState implements the loader state.
type loaderState struct {
	parsers  map[string]core.LoaderParserModule
	store    core.Store
	fetcher  core.Fetcher
	mediator *loaderMediator
	failsafe bool
}

const (
	loaderModuleID module.ModuleID = "app.loader"

	loaderConfigDefaultExecStartup          int = 15
	loaderConfigDefaultExecInterval         int = 900
	loaderConfigDefaultExecFailsafeInterval int = 300
	loaderConfigDefaultExecWorkers          int = 1
	loaderConfigDefaultExecMaxOps           int = 100
	loaderConfigDefaultExecMaxDelay         int = 60
)

// ModuleInfo returns the module information.
func (l loader) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           loaderModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &loader{
				logger: slog.New(log.NewHandler(os.Stderr, string(loaderModuleID), nil)),
				state: &loaderState{
					parsers: make(map[string]core.LoaderParserModule),
				},
				mu:   &sync.RWMutex{},
				stop: make(chan struct{}),
			}
		},
	}
}

// Init initializes the loader.
func (l *loader) Init(config map[string]interface{}) error {
	l.logger.Debug("Initializing loader")

	if config == nil {
		l.config = &loaderConfig{}
	} else {
		if err := mapstructure.Decode(config, &l.config); err != nil {
			l.logger.Error("Failed to parse configuration", "err", err)
			return fmt.Errorf("parse config: %w", err)
		}
	}

	var errConfig bool

	if l.config.ExecStartup == nil {
		defaultValue := loaderConfigDefaultExecStartup
		l.config.ExecStartup = &defaultValue
	}
	if *l.config.ExecStartup < 0 {
		l.logger.Error("Invalid value", "option", "ExecStartup", "value", *l.config.ExecStartup)
		errConfig = true
	}
	if l.config.ExecInterval == nil {
		defaultValue := loaderConfigDefaultExecInterval
		l.config.ExecInterval = &defaultValue
	}
	if *l.config.ExecInterval < 0 {
		l.logger.Error("Invalid value", "option", "ExecInterval", "value", *l.config.ExecInterval)
		errConfig = true
	}
	if l.config.ExecFailsafeInterval == nil {
		defaultValue := loaderConfigDefaultExecFailsafeInterval
		l.config.ExecFailsafeInterval = &defaultValue
	}
	if *l.config.ExecFailsafeInterval < 0 {
		l.logger.Error("Invalid value", "option", "ExecFailsafeInterval", "value", *l.config.ExecFailsafeInterval)
		errConfig = true
	}
	if l.config.ExecWorkers == nil {
		defaultValue := loaderConfigDefaultExecWorkers
		l.config.ExecWorkers = &defaultValue
	}
	if *l.config.ExecWorkers <= 0 {
		l.logger.Error("Invalid value", "option", "ExecWorkers", "value", *l.config.ExecWorkers)
		errConfig = true
	}
	if l.config.ExecMaxOps == nil {
		defaultValue := loaderConfigDefaultExecMaxOps
		l.config.ExecMaxOps = &defaultValue
	}
	if *l.config.ExecMaxOps < 0 {
		l.logger.Error("Invalid value", "option", "ExecMaxOps", "value", *l.config.ExecMaxOps)
		errConfig = true
	}
	if l.config.ExecMaxDelay == nil {
		defaultValue := loaderConfigDefaultExecMaxDelay
		l.config.ExecMaxDelay = &defaultValue
	}
	if *l.config.ExecMaxDelay < 0 {
		l.logger.Error("Invalid value", "option", "ExecMaxDelay", "value", *l.config.ExecMaxDelay)
		errConfig = true
	}

	for ruleName, ruleConfig := range l.config.Rules {
		for moduleName, moduleConfig := range ruleConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("app.loader.parser." + moduleName))
			if err != nil {
				l.logger.Error("Unregistered parser module", "rule", ruleName, "module", moduleName, "err", err)
				errConfig = true
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.LoaderParserModule)
			if !ok {
				err := errors.New("module instance not valid")
				l.logger.Error("Invalid parser module", "rule", ruleName, "module", moduleName, "err", err)
				errConfig = true
				continue
			}

			if moduleConfig == nil {
				moduleConfig = map[string]interface{}{}
			}
			if err := module.Init(moduleConfig); err != nil {
				l.logger.Error("Failed to init parser module", "rule", ruleName, "module", moduleName, "err", err)
				errConfig = true
				continue
			}

			l.state.parsers[ruleName] = module

			break
		}
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the loader.
func (l *loader) Register(app core.App) error {
	l.state.mediator = newLoaderMediator(l)
	l.state.store = app.Store()
	l.state.fetcher = app.Fetcher()

	return nil
}

// Start starts the loader.
func (l *loader) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.config.Rules) > 0 {
		if *l.config.ExecStartup == 0 && *l.config.ExecInterval == 0 {
			l.logger.Warn("Periodic execution disabled")
		}
		if *l.config.ExecStartup > 0 && *l.config.ExecInterval == 0 {
			l.logger.Warn("One-off execution enabled")
		}
		if (*l.config.ExecStartup > 0 || *l.config.ExecInterval > 0) && *l.config.ExecFailsafeInterval == 0 {
			l.logger.Warn("Failsafe execution disabled")
		}

		if *l.config.ExecStartup > 0 || *l.config.ExecInterval > 0 {
			l.logger.Info("Starting loader")

			l.execute(l.stop)
		}
	}

	return nil
}

// Stop stops the loader.
func (l *loader) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.config.Rules) > 0 && (*l.config.ExecStartup > 0 || *l.config.ExecInterval > 0) {
		l.logger.Info("Stopping loader")

		l.stop <- struct{}{}
	}

	return nil
}

// execute loads all resources data.
func (l *loader) execute(stop <-chan struct{}) {
	startup := true
	var delay time.Duration
	if *l.config.ExecStartup > 0 {
		delay = time.Duration(*l.config.ExecStartup) * time.Second
	} else {
		delay = time.Duration(*l.config.ExecInterval) * time.Second
	}
	ticker := time.NewTicker(delay)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		worker := func(ctx context.Context, jobs <-chan string, results chan<- error) {
			for ruleName := range jobs {
				parser, ok := l.state.parsers[ruleName]
				if !ok {
					err := errors.New("parser not found")
					l.logger.Error("Execution error", "rule", ruleName, "err", err)
					results <- err
					continue
				}
				if err := parser.Parse(ctx, l.state.store, l.state.fetcher); err != nil {
					l.logger.Error("Execution error", "rule", ruleName, "err", err)
					results <- err
					continue
				}
				results <- nil
			}
		}

	loop:
		for {
			select {
			case <-stop:
				l.logger.Debug("New stop event received, exiting")
				break loop

			case <-ticker.C:
				startTime := time.Now()

				l.logger.Debug("Starting new execution")

				if startup {
					startup = false
					if *l.config.ExecInterval > 0 {
						ticker.Reset(time.Duration(*l.config.ExecInterval) * time.Second)
					} else {
						ticker.Stop()
					}
				}

				rulesCount := len(l.config.Rules)
				jobs := make(chan string, rulesCount)
				results := make(chan error, rulesCount)

				for w := 1; w <= *l.config.ExecWorkers; w++ {
					go worker(ctx, jobs, results)
				}

				ops := 0

				for ruleName := range l.config.Rules {
					ops += 1

					if *l.config.ExecMaxOps > 0 && ops > *l.config.ExecMaxOps {
						l.logger.Warn("Max operations per execution reached, delaying execution", "delay", l.config.ExecMaxDelay)

						time.Sleep(time.Duration(*l.config.ExecMaxDelay) * time.Second)
						ops = 1
					}

					jobs <- ruleName
				}

				close(jobs)

				success := 0
				failure := 0

				for job := 1; job <= rulesCount; job++ {
					select {
					case <-stop:
						break loop
					case <-ctx.Done():
						break loop
					case err := <-results:
						if err != nil {
							failure += 1
						} else {
							success += 1
						}
					}
				}

				l.logger.Info("Execution done", "total", rulesCount, "success", success, "failure", failure,
					"duration", time.Since(startTime).Round(time.Second))

				if failure > 0 && !l.state.failsafe && *l.config.ExecFailsafeInterval > 0 {
					l.logger.Warn("Last execution failed, enabling failsafe mode")

					ticker.Reset(time.Duration(*l.config.ExecFailsafeInterval) * time.Second)
					l.state.failsafe = true
				}
				if failure == 0 && l.state.failsafe {
					l.logger.Warn("Last execution succeeded, disabling failsafe mode")

					if *l.config.ExecInterval > 0 {
						ticker.Reset(time.Duration(*l.config.ExecInterval) * time.Second)
					} else {
						ticker.Stop()
					}
					l.state.failsafe = false
				}
			}
		}

		ticker.Stop()
	}()
}

var _ Loader = (*loader)(nil)

// loaderMediator implements the loader mediator.
type loaderMediator struct {
	loader *loader
}

// newLoaderMediator creates a new mediator.
func newLoaderMediator(loader *loader) *loaderMediator {
	return &loaderMediator{
		loader: loader,
	}
}

var _ core.Loader = (*loaderMediator)(nil)
