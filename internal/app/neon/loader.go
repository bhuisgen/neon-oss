// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// loader implements the loader.
type loader struct {
	config  *loaderConfig
	logger  *log.Logger
	state   *loaderState
	store   Store
	fetcher Fetcher
	mu      sync.RWMutex
	stop    chan struct{}
}

// loaderConfig implements the loader configuration.
type loaderConfig struct {
	ExecStartup          *int
	ExecInterval         *int
	ExecFailsafeInterval *int
	ExecWorkers          *int
	ExecMaxOps           *int
	ExecMaxDelay         *int
	Rules                map[string]map[string]map[string]interface{}
}

// loaderState implements the loader state.
type loaderState struct {
	parsers  map[string]core.LoaderParserModule
	failsafe bool
}

const (
	loaderLogger string = "loader"

	loaderConfigDefaultExecStartup          int = 15
	loaderConfigDefaultExecInterval         int = 900
	loaderConfigDefaultExecFailsafeInterval int = 300
	loaderConfigDefaultExecWorkers          int = 1
	loaderConfigDefaultExecMaxOps           int = 100
	loaderConfigDefaultExecMaxDelay         int = 60
)

// newLoader creates a new loader.
func newLoader(store Store, fetcher Fetcher) *loader {
	return &loader{
		logger: log.New(os.Stderr, fmt.Sprint(loaderLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		state: &loaderState{
			parsers: make(map[string]core.LoaderParserModule),
		},
		store:   store,
		fetcher: fetcher,
		stop:    make(chan struct{}),
	}
}

// Init initializes the loader.
func (l *loader) Init(config map[string]interface{}) error {
	if config == nil {
		l.config = &loaderConfig{}
	} else {
		if err := mapstructure.Decode(config, &l.config); err != nil {
			l.logger.Printf("failed to parse configuration")
			return err
		}
	}

	var errInit bool

	if l.config.ExecStartup == nil {
		defaultValue := loaderConfigDefaultExecStartup
		l.config.ExecStartup = &defaultValue
	}
	if *l.config.ExecStartup < 0 {
		l.logger.Printf("option '%s', invalid value '%d'", "ExecStartup", *l.config.ExecStartup)
		errInit = true
	}
	if l.config.ExecInterval == nil {
		defaultValue := loaderConfigDefaultExecInterval
		l.config.ExecInterval = &defaultValue
	}
	if *l.config.ExecInterval < 0 {
		l.logger.Printf("option '%s', invalid value '%d'", "ExecInterval", *l.config.ExecInterval)
		errInit = true
	}
	if l.config.ExecFailsafeInterval == nil {
		defaultValue := loaderConfigDefaultExecFailsafeInterval
		l.config.ExecFailsafeInterval = &defaultValue
	}
	if *l.config.ExecFailsafeInterval < 0 {
		l.logger.Printf("option '%s', invalid value '%d'", "ExecFailsafeInterval", *l.config.ExecFailsafeInterval)
		errInit = true
	}
	if l.config.ExecWorkers == nil {
		defaultValue := loaderConfigDefaultExecWorkers
		l.config.ExecWorkers = &defaultValue
	}
	if *l.config.ExecWorkers < 0 {
		l.logger.Printf("option '%s', invalid value '%d'", "ExecWorkers", *l.config.ExecWorkers)
		errInit = true
	}
	if l.config.ExecMaxOps == nil {
		defaultValue := loaderConfigDefaultExecMaxOps
		l.config.ExecMaxOps = &defaultValue
	}
	if *l.config.ExecMaxOps < 0 {
		l.logger.Printf("option '%s', invalid value '%d'", "ExecMaxOps", *l.config.ExecMaxOps)
		errInit = true
	}
	if l.config.ExecMaxDelay == nil {
		defaultValue := loaderConfigDefaultExecMaxDelay
		l.config.ExecMaxDelay = &defaultValue
	}
	if *l.config.ExecMaxDelay < 0 {
		l.logger.Printf("option '%s', invalid value '%d'", "ExecMaxDelay", *l.config.ExecMaxDelay)
		errInit = true
	}

	for ruleName, ruleConfig := range l.config.Rules {
		for moduleName, moduleConfig := range ruleConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("loader.parser." + moduleName))
			if err != nil {
				l.logger.Printf("rule '%s', unregistered parser module '%s'", ruleName, moduleName)
				errInit = true
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.LoaderParserModule)
			if !ok {
				l.logger.Printf("rule '%s', invalid parser module '%s'", ruleName, moduleName)
				errInit = true
				continue
			}

			if moduleConfig == nil {
				moduleConfig = map[string]interface{}{}
			}
			if err := module.Init(
				moduleConfig,
				log.New(os.Stderr, fmt.Sprint(l.logger.Prefix(), "parser[", moduleName, "]: "),
					log.LstdFlags|log.Lmsgprefix),
			); err != nil {
				l.logger.Printf("rule '%s', failed to init parser module '%s'", ruleName, moduleName)
				errInit = true
				continue
			}

			l.state.parsers[ruleName] = module

			break
		}
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Start starts the loader.
func (l *loader) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if *l.config.ExecInterval > 0 {
		l.execute(l.stop)
	}

	return nil
}

// Stop stops the loader.
func (l *loader) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if *l.config.ExecInterval > 0 {
		l.stop <- struct{}{}
	}

	return nil
}

// execute loads all resources data.
func (l *loader) execute(stop <-chan struct{}) {
	startup := true
	ticker := time.NewTicker(time.Duration(*l.config.ExecStartup) * time.Second)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		worker := func(ctx context.Context, id int, jobs <-chan string, results chan<- error) {
			for ruleName := range jobs {
				parser, ok := l.state.parsers[ruleName]
				if !ok {
					err := fmt.Errorf("parser module not found")
					l.logger.Printf("rule '%s', parser error: %s", ruleName, err)
					results <- err
					continue
				}

				err := parser.Parse(ctx, l.store, l.fetcher)
				if err != nil {
					l.logger.Printf("rule '%s', parser error: %s", ruleName, err)
				}
				results <- err
			}
		}

	loop:
		for {
			select {
			case <-stop:
				break loop

			case <-ticker.C:
				if startup {
					startup = false

					ticker.Stop()
					ticker = time.NewTicker(time.Duration(*l.config.ExecInterval) * time.Second)
				}

				rulesCount := len(l.config.Rules)
				jobs := make(chan string, rulesCount)
				results := make(chan error, rulesCount)

				for w := 1; w <= *l.config.ExecWorkers; w++ {
					go worker(ctx, w, jobs, results)
				}

				ops := 0

				for ruleName := range l.config.Rules {
					ops += 1

					if *l.config.ExecMaxOps > 0 && ops > *l.config.ExecMaxOps {
						l.logger.Printf("Max operations per execution reached, delaying execution for %d seconds",
							l.config.ExecMaxDelay)

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

				l.logger.Printf("Execution finished (success=%d, failure=%d, total=%d)", success, failure, rulesCount)

				if failure > 0 && !l.state.failsafe && *l.config.ExecFailsafeInterval > 0 {
					l.logger.Print("Last execution failed, enabling failsafe mode")

					ticker.Stop()
					ticker = time.NewTicker(time.Duration(*l.config.ExecFailsafeInterval) * time.Second)
				}
				if failure == 0 && l.state.failsafe {
					l.logger.Print("Last execution succeeded, disabling failsafe mode")

					ticker.Stop()
					ticker = time.NewTicker(time.Duration(*l.config.ExecInterval) * time.Second)
				}
				if failure == 0 {
					l.state.failsafe = false
				} else {
					l.state.failsafe = true
				}
			}
		}

		ticker.Stop()
	}()
}

var _ Loader = (*loader)(nil)
