// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
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
	parsersModules map[string]core.LoaderParserModule
	failsafe       bool
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
		store:   store,
		fetcher: fetcher,
	}
}

// Check checks the loader configuration.
func (l *loader) Check(config map[string]interface{}) ([]string, error) {
	l.state = &loaderState{}

	var report []string

	var c loaderConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "loader: failed to parse configuration")
		return report, err
	}

	if c.ExecStartup == nil {
		defaultValue := loaderConfigDefaultExecStartup
		c.ExecStartup = &defaultValue
	}
	if *c.ExecStartup < 0 {
		report = append(report, fmt.Sprintf("loader: option '%s', invalid value '%d'", "ExecStartup", *c.ExecStartup))
	}
	if c.ExecInterval == nil {
		defaultValue := loaderConfigDefaultExecInterval
		c.ExecInterval = &defaultValue
	}
	if *c.ExecInterval < 0 {
		report = append(report, fmt.Sprintf("loader: option '%s', invalid value '%d'", "ExecInterval", *c.ExecInterval))
	}
	if c.ExecFailsafeInterval == nil {
		defaultValue := loaderConfigDefaultExecFailsafeInterval
		c.ExecFailsafeInterval = &defaultValue
	}
	if *c.ExecFailsafeInterval < 0 {
		report = append(report, fmt.Sprintf("loader: option '%s', invalid value '%d'", "ExecFailsafeInterval",
			*c.ExecFailsafeInterval))
	}
	if c.ExecWorkers == nil {
		defaultValue := loaderConfigDefaultExecWorkers
		c.ExecWorkers = &defaultValue
	}
	if *c.ExecWorkers < 0 {
		report = append(report, fmt.Sprintf("loader: option '%s', invalid value '%d'", "ExecWorkers", *c.ExecWorkers))
	}
	if c.ExecMaxOps == nil {
		defaultValue := loaderConfigDefaultExecMaxOps
		c.ExecMaxOps = &defaultValue
	}
	if *c.ExecMaxOps < 0 {
		report = append(report, fmt.Sprintf("loader: option '%s', invalid value '%d'", "ExecMaxOps", *c.ExecMaxOps))
	}
	if c.ExecMaxDelay == nil {
		defaultValue := loaderConfigDefaultExecMaxDelay
		c.ExecMaxDelay = &defaultValue
	}
	if *c.ExecMaxDelay < 0 {
		report = append(report, fmt.Sprintf("loader: option '%s', invalid value '%d'", "ExecMaxDelay", *c.ExecMaxDelay))
	}
	for ruleName, ruleConfig := range c.Rules {
		for moduleName, moduleConfig := range ruleConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("loader.parser." + moduleName))
			if err != nil {
				report = append(report, fmt.Sprintf("loader: rule '%s', unregistered parser module '%s'", ruleName,
					moduleName))
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.LoaderParserModule)
			if !ok {
				report = append(report, fmt.Sprintf("loader: rule '%s', invalid parser module '%s'", ruleName, moduleName))
				continue
			}
			r, err := module.Check(moduleConfig)
			if err != nil {
				for _, line := range r {
					report = append(report, fmt.Sprintf("loader: rule '%s', failed to check configuration: %s", ruleName, line))
				}
				continue
			}

			break
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the loader.
func (l *loader) Load(config map[string]interface{}) error {
	var c loaderConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	l.config = &c
	l.logger = log.New(os.Stderr, fmt.Sprint(loaderLogger, ": "), log.LstdFlags|log.Lmsgprefix)
	l.state = &loaderState{
		parsersModules: make(map[string]core.LoaderParserModule),
	}
	l.stop = make(chan struct{})

	if l.config.ExecStartup == nil {
		defaultValue := loaderConfigDefaultExecStartup
		l.config.ExecStartup = &defaultValue
	}
	if l.config.ExecInterval == nil {
		defaultValue := loaderConfigDefaultExecInterval
		l.config.ExecInterval = &defaultValue
	}
	if l.config.ExecFailsafeInterval == nil {
		defaultValue := loaderConfigDefaultExecFailsafeInterval
		l.config.ExecFailsafeInterval = &defaultValue
	}
	if l.config.ExecWorkers == nil {
		defaultValue := loaderConfigDefaultExecWorkers
		l.config.ExecWorkers = &defaultValue
	}
	if l.config.ExecMaxOps == nil {
		defaultValue := loaderConfigDefaultExecMaxOps
		l.config.ExecMaxOps = &defaultValue
	}
	if l.config.ExecMaxDelay == nil {
		defaultValue := loaderConfigDefaultExecMaxDelay
		l.config.ExecMaxDelay = &defaultValue
	}
	for ruleName, ruleConfig := range c.Rules {
		for moduleName, moduleConfig := range ruleConfig {
			moduleInfo, err := module.Lookup(module.ModuleID("loader.parser." + moduleName))
			if err != nil {
				return err
			}
			module, ok := moduleInfo.NewInstance().(core.LoaderParserModule)
			if !ok {
				return fmt.Errorf("rule '%s', invalid parser module '%s'", ruleName, moduleName)
			}
			err = module.Load(moduleConfig)
			if err != nil {
				return err
			}

			l.state.parsersModules[ruleName] = module

			break
		}
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
				parser, ok := l.state.parsersModules[ruleName]
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

				l.logger.Printf("Results: success=%d, failure=%d, total=%d", success, failure, rulesCount)

				if failure > 0 && !l.state.failsafe && *l.config.ExecFailsafeInterval > 0 {
					l.logger.Print("Last execution failed, failsafe mode enabled")

					ticker.Stop()
					ticker = time.NewTicker(time.Duration(*l.config.ExecFailsafeInterval) * time.Second)
				}
				if failure == 0 && l.state.failsafe {
					l.logger.Print("Last execution recovered, failsafe mode disabled")

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
