// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// loader implements a loader
type loader struct {
	config   *LoaderConfig
	logger   *log.Logger
	executor LoaderExecutor
	stop     chan struct{}
}

// loaderExecutor implements a loader executor
type loaderExecutor struct {
	config        *LoaderConfig
	logger        *log.Logger
	fetcher       Fetcher
	jsonUnmarshal func(data []byte, v any) error
}

// LoaderConfig implements the loader configuration
type LoaderConfig struct {
	ExecStartup  int
	ExecInterval int
	ExecWorkers  int
	Rules        []LoaderRule
}

// LoaderRule implements a rule
type LoaderRule struct {
	Name   string
	Type   string
	Static LoaderRuleStatic
	Single LoaderRuleSingle
	List   LoaderRuleList
}

// LoaderRuleStatic implements a static rule
type LoaderRuleStatic struct {
	Resource string
}

// LoaderRuleSingle implements a single rule
type LoaderRuleSingle struct {
	Resource                    string
	ResourcePayloadItem         string
	ItemTemplate                string
	ItemTemplateResource        string
	ItemTemplateResourceParams  map[string]string
	ItemTemplateResourceHeaders map[string]string
}

// LoaderRuleList implements a list rule
type LoaderRuleList struct {
	Resource                    string
	ResourcePayloadItems        string
	ItemTemplate                string
	ItemTemplateResource        string
	ItemTemplateResourceParams  map[string]string
	ItemTemplateResourceHeaders map[string]string
}

const (
	loaderLogger     string = "loader"
	loaderTypeStatic string = "static"
	loaderTypeSingle string = "single"
	loaderTypeList   string = "list"
)

// loaderJsonUnmarshal redirects to json.Unmarshal
func loaderJsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// CreateLoader creates a new loader instance
func CreateLoader(config *LoaderConfig, fetcher Fetcher) (*loader, error) {
	logger := log.New(os.Stderr, fmt.Sprint(loaderLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	return &loader{
		config: config,
		logger: logger,
		executor: loaderExecutor{
			config:        config,
			logger:        logger,
			fetcher:       fetcher,
			jsonUnmarshal: loaderJsonUnmarshal,
		},
		stop: make(chan struct{}),
	}, nil
}

// Start starts the loader
func (l *loader) Start() error {
	if l.config.ExecInterval > 0 {
		l.executor.execute(l.stop)
	}

	return nil
}

// Stop stops the loader
func (l *loader) Stop() error {
	if l.config.ExecInterval > 0 {
		l.stop <- struct{}{}
	}

	return nil
}

// execute loads all resources data
func (e loaderExecutor) execute(stop <-chan struct{}) {
	startup := true
	ticker := time.NewTicker(time.Duration(e.config.ExecStartup) * time.Second)

	go func() {
		ctx := context.Background()

		worker := func(ctx context.Context, id int, jobs <-chan int, results chan<- error) {
			for ruleIndex := range jobs {
				rule := e.config.Rules[ruleIndex]
				var err error
				switch rule.Type {
				case loaderTypeStatic:
					err = e.loadStatic(ctx, &rule.Static)
				case loaderTypeSingle:
					err = e.loadSingle(ctx, &rule.Single)
				case loaderTypeList:
					err = e.loadList(ctx, &rule.List)
				}

				results <- err
			}
		}

		for {
			select {
			case <-ticker.C:
				if startup {
					startup = false

					ticker.Stop()
					ticker = time.NewTicker(time.Duration(e.config.ExecInterval) * time.Second)
				}

				jobsCount := len(e.config.Rules)
				jobs := make(chan int, jobsCount)
				results := make(chan error, jobsCount)

				for w := 1; w <= e.config.ExecWorkers; w++ {
					go worker(ctx, w, jobs, results)
				}

				for ruleIndex := range e.config.Rules {
					jobs <- ruleIndex
				}

				close(jobs)

				success := 0
				failure := 0
				for job := 1; job <= jobsCount; job++ {
					select {
					case err := <-results:
						if err != nil {
							failure += 1
							continue
						}
						success += 1

					case <-ctx.Done():
						results <- ctx.Err()
					}
				}

				e.logger.Printf("Results: success=%d, failure=%d, total=%d", success, failure, jobsCount)

			case <-stop:
				ticker.Stop()

				return
			}
		}
	}()
}

// loadStatic loads a static resource
func (e loaderExecutor) loadStatic(ctx context.Context, rule *LoaderRuleStatic) error {
	err := e.fetcher.Fetch(ctx, rule.Resource)
	if err != nil {
		return err
	}

	return nil
}

// loadSingle loads a single resource
func (e loaderExecutor) loadSingle(ctx context.Context, rule *LoaderRuleSingle) error {
	err := e.fetcher.Fetch(ctx, rule.Resource)
	if err != nil {
		return err
	}

	response, err := e.fetcher.Get(rule.Resource)
	if err != nil {
		return err
	}

	var payload interface{}
	err = e.jsonUnmarshal(response, &payload)
	if err != nil {
		return err
	}
	mPayload := payload.(map[string]interface{})
	responseData := mPayload[rule.ResourcePayloadItem]
	item := responseData.(map[string]interface{})
	mItem := make(map[string]string)
	for k, v := range item {
		switch value := v.(type) {
		case string:
			mItem[k] = value
		case float64:
			mItem[k] = strconv.FormatFloat(value, 'f', -1, 64)
		case bool:
			mItem[k] = strconv.FormatBool(value)
		}
	}

	rKey := replaceLoaderResourceParameters(rule.ItemTemplateResource, mItem)

	var rParams map[string]string
	for rParamKey, rParamValue := range rule.ItemTemplateResourceParams {
		if rParams == nil {
			rParams = make(map[string]string)
		}
		rParamKey = replaceLoaderResourceParameters(rParamKey, mItem)
		rParamValue = replaceLoaderResourceParameters(rParamValue, mItem)
		rParams[rParamKey] = rParamValue
	}

	var rHeaders map[string]string
	for rHeaderKey, rHeaderValue := range rule.ItemTemplateResourceParams {
		if rHeaders == nil {
			rHeaders = make(map[string]string)
		}
		rHeaderKey = replaceLoaderResourceParameters(rHeaderKey, mItem)
		rHeaderValue = replaceLoaderResourceParameters(rHeaderValue, mItem)
		rHeaders[rHeaderKey] = rHeaderValue
	}

	r, err := e.fetcher.CreateResourceFromTemplate(rule.ItemTemplate, rKey, rParams, rHeaders)
	if err != nil {
		return err
	}

	if !e.fetcher.Exists(r.Name) {
		e.fetcher.Register(*r)
	}

	err = e.fetcher.Fetch(ctx, r.Name)
	if err != nil {
		return err
	}

	return nil
}

// loadList loads a list resource
func (e loaderExecutor) loadList(ctx context.Context, rule *LoaderRuleList) error {
	err := e.fetcher.Fetch(ctx, rule.Resource)
	if err != nil {
		return err
	}

	response, err := e.fetcher.Get(rule.Resource)
	if err != nil {
		return err
	}

	var payload interface{}
	err = e.jsonUnmarshal(response, &payload)
	if err != nil {
		return err
	}
	mPayload := payload.(map[string]interface{})
	responseData := mPayload[rule.ResourcePayloadItems]
	for _, data := range responseData.([]interface{}) {
		item := data.(map[string]interface{})
		mItem := make(map[string]string)
		for k, v := range item {
			switch value := v.(type) {
			case string:
				mItem[k] = value
			case float64:
				mItem[k] = strconv.FormatFloat(value, 'f', -1, 64)
			case bool:
				mItem[k] = strconv.FormatBool(value)
			}
		}

		rKey := replaceLoaderResourceParameters(rule.ItemTemplateResource, mItem)

		var rParams map[string]string
		for rParamKey, rParamValue := range rule.ItemTemplateResourceParams {
			if rParams == nil {
				rParams = make(map[string]string)
			}
			rParamKey = replaceLoaderResourceParameters(rParamKey, mItem)
			rParamValue = replaceLoaderResourceParameters(rParamValue, mItem)
			rParams[rParamKey] = rParamValue
		}

		var rHeaders map[string]string
		for rHeaderKey, rHeaderValue := range rule.ItemTemplateResourceParams {
			if rHeaders == nil {
				rHeaders = make(map[string]string)
			}
			rHeaderKey = replaceLoaderResourceParameters(rHeaderKey, mItem)
			rHeaderValue = replaceLoaderResourceParameters(rHeaderValue, mItem)
			rHeaders[rHeaderKey] = rHeaderValue
		}

		r, err := e.fetcher.CreateResourceFromTemplate(rule.ItemTemplate, rKey, rParams, rHeaders)
		if err != nil {
			return err
		}

		if !e.fetcher.Exists(r.Name) {
			e.fetcher.Register(*r)
		}

		err = e.fetcher.Fetch(ctx, r.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// replaceLoaderResourceParameters returns a copy of the string s with all its parameters replaced
func replaceLoaderResourceParameters(s string, params map[string]string) string {
	tmp := s
	for key, value := range params {
		tmp = strings.ReplaceAll(tmp, fmt.Sprint("$", key), value)
	}
	return tmp
}
