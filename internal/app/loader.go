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

// loader implements the resources loader
type loader struct {
	config  *LoaderConfig
	logger  *log.Logger
	stop    chan struct{}
	fetcher *fetcher
}

// LoaderConfig implements the resources loader configuration
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

// NewLoader creates a new loader
func NewLoader(config *LoaderConfig, fetcher *fetcher) *loader {
	logger := log.New(os.Stdout, fmt.Sprint(loaderLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	return &loader{
		config:  config,
		logger:  logger,
		stop:    make(chan struct{}),
		fetcher: fetcher,
	}
}

// Start starts the loader
func (l *loader) Start() {
	if l.config.ExecInterval > 0 {
		l.execute()
	}
}

// Start stops the loader
func (l *loader) Stop() {
	if l.config.ExecInterval > 0 {
		l.stop <- struct{}{}
	}
}

// execute loads all resources data
func (l *loader) execute() {
	startup := true
	ticker := time.NewTicker(time.Duration(l.config.ExecStartup) * time.Second)

	go func() {
		ctx := context.Background()

		worker := func(ctx context.Context, id int, jobs <-chan int, results chan<- error) {
			for ruleIndex := range jobs {
				rule := l.config.Rules[ruleIndex]
				var err error
				switch rule.Type {
				case loaderTypeStatic:
					err = l.loadStatic(ctx, &rule.Static)
				case loaderTypeSingle:
					err = l.loadSingle(ctx, &rule.Single)
				case loaderTypeList:
					err = l.loadList(ctx, &rule.List)
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
					ticker = time.NewTicker(time.Duration(l.config.ExecInterval) * time.Second)
				}

				jobsCount := len(l.config.Rules)
				jobs := make(chan int, jobsCount)
				results := make(chan error, jobsCount)

				for w := 1; w <= l.config.ExecWorkers; w++ {
					go worker(ctx, w, jobs, results)
				}

				for ruleIndex := range l.config.Rules {
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

				l.logger.Printf("Results: success=%d, failure=%d, total=%d", success, failure, jobsCount)

			case <-l.stop:
				ticker.Stop()

				return
			}
		}
	}()
}

// loadStatic loads a static resource
func (l *loader) loadStatic(ctx context.Context, rule *LoaderRuleStatic) error {
	err := l.fetcher.Fetch(ctx, rule.Resource)
	if err != nil {
		return err
	}

	return nil
}

// loadSingle loads a single resource
func (l *loader) loadSingle(ctx context.Context, rule *LoaderRuleSingle) error {
	err := l.fetcher.Fetch(ctx, rule.Resource)
	if err != nil {
		return err
	}

	response, err := l.fetcher.Get(rule.Resource)
	if err != nil {
		return err
	}

	var payload interface{}
	err = json.Unmarshal(response, &payload)
	if err != nil {
		return err
	}
	mPayload := payload.(map[string]interface{})
	responseData := mPayload[rule.ResourcePayloadItem]
	item := responseData.(map[string]interface{})
	mItem := make(map[string]string)
	for k, v := range item {
		switch v.(type) {
		case string:
			mItem[k] = v.(string)
		case int64:
			mItem[k] = strconv.FormatInt(v.(int64), 10)
		case bool:
			mItem[k] = strconv.FormatBool(v.(bool))
		}
	}

	rKey := replaceLoaderResourceParameters(rule.ItemTemplateResource, mItem)
	rParams := make(map[string]string)
	for rParamKey, rParamValue := range rule.ItemTemplateResourceParams {
		rParamKey = replaceLoaderResourceParameters(rParamKey, mItem)
		rParamValue = replaceLoaderResourceParameters(rParamValue, mItem)
		rParams[rParamKey] = rParamValue
	}

	rHeaders := make(map[string]string)
	for rHeaderKey, rHeaderValue := range rule.ItemTemplateResourceParams {
		rHeaderKey = replaceLoaderResourceParameters(rHeaderKey, mItem)
		rHeaderValue = replaceLoaderResourceParameters(rHeaderValue, mItem)
		rHeaders[rHeaderKey] = rHeaderValue
	}

	r, err := l.fetcher.CreateResourceFromTemplate(rule.ItemTemplate, rKey, rParams, rHeaders)
	if err != nil {
		return err
	}

	if !l.fetcher.Exists(r.Name) {
		l.fetcher.Register(r)
	}

	err = l.fetcher.Fetch(ctx, r.Name)
	if err != nil {
		return err
	}

	return nil
}

// loadList loads a list resource
func (l *loader) loadList(ctx context.Context, rule *LoaderRuleList) error {
	err := l.fetcher.Fetch(ctx, rule.Resource)
	if err != nil {
		return err
	}

	response, err := l.fetcher.Get(rule.Resource)
	if err != nil {
		return err
	}

	var payload interface{}
	err = json.Unmarshal(response, &payload)
	if err != nil {
		return err
	}
	mPayload := payload.(map[string]interface{})
	responseData := mPayload[rule.ResourcePayloadItems]
	for _, data := range responseData.([]interface{}) {
		item := data.(map[string]interface{})
		mItem := make(map[string]string)
		for k, v := range item {
			switch v.(type) {
			case string:
				mItem[k] = v.(string)
			case int64:
				mItem[k] = strconv.FormatInt(v.(int64), 10)
			case bool:
				mItem[k] = strconv.FormatBool(v.(bool))
			}
		}

		rKey := replaceLoaderResourceParameters(rule.ItemTemplateResource, mItem)
		rParams := make(map[string]string)
		for rParamKey, rParamValue := range rule.ItemTemplateResourceParams {
			rParamKey = replaceLoaderResourceParameters(rParamKey, mItem)
			rParamValue = replaceLoaderResourceParameters(rParamValue, mItem)
			rParams[rParamKey] = rParamValue
		}

		rHeaders := make(map[string]string)
		for rHeaderKey, rHeaderValue := range rule.ItemTemplateResourceParams {
			rHeaderKey = replaceLoaderResourceParameters(rHeaderKey, mItem)
			rHeaderValue = replaceLoaderResourceParameters(rHeaderValue, mItem)
			rHeaders[rHeaderKey] = rHeaderValue
		}

		r, err := l.fetcher.CreateResourceFromTemplate(rule.ItemTemplate, rKey, rParams, rHeaders)
		if err != nil {
			return err
		}

		if !l.fetcher.Exists(r.Name) {
			l.fetcher.Register(r)
		}

		err = l.fetcher.Fetch(ctx, r.Name)
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
