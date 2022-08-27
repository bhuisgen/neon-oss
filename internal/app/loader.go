// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"
)

// loader implements the resources loader
type loader struct {
	config       *LoaderConfig
	logger       *log.Logger
	dataFailsafe bool
	stopData     chan struct{}
	fetcher      *fetcher
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
	Resource              string
	ResourcePayloadItem   string
	ItemTemplate          string
	ItemTemplateKey       string
	ItemTemplateKeyParams map[string]string
}

// LoaderRuleList implements a list rule
type LoaderRuleList struct {
	Resource              string
	ResourcePayloadItems  string
	ItemTemplate          string
	ItemTemplateKey       string
	ItemTemplateKeyParams map[string]string
}

const (
	LOADER_TYPE_STATIC string = "static"
	LOADER_TYPE_SINGLE string = "single"
	LOADER_TYPE_LIST   string = "list"
)

// NewLoader creates a new loader
func NewLoader(config *LoaderConfig, fetcher *fetcher) *loader {
	return &loader{
		config:       config,
		logger:       log.Default(),
		dataFailsafe: false,
		stopData:     make(chan struct{}),
		fetcher:      fetcher,
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
		l.stopData <- struct{}{}
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
				case LOADER_TYPE_STATIC:
					err = l.loadStatic(ctx, rule.Static.Resource)
				case LOADER_TYPE_SINGLE:
					err = l.loadSingle(ctx, rule.Single.Resource, rule.Single.ResourcePayloadItem, rule.Single.ItemTemplate,
						rule.Single.ItemTemplateKey, rule.Single.ItemTemplateKeyParams)
				case LOADER_TYPE_LIST:
					err = l.loadList(ctx, rule.List.Resource, rule.List.ResourcePayloadItems, rule.List.ItemTemplate,
						rule.List.ItemTemplateKey, rule.List.ItemTemplateKeyParams)
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

				l.logger.Printf("Loader data results: success=%d, failure=%d, total=%d", success, failure,
					jobsCount)

			case <-l.stopData:
				ticker.Stop()

				return
			}
		}
	}()
}

// loadStatic loads a static resource
func (l *loader) loadStatic(ctx context.Context, resource string) error {
	err := l.fetcher.Fetch(ctx, resource)
	if err != nil {
		return err
	}

	return nil
}

// loadSingle loads a single resource
func (l *loader) loadSingle(ctx context.Context, resource string, payloadItem string, itemTemplate string,
	itemKey string, itemKeyParams map[string]string) error {
	err := l.fetcher.Fetch(ctx, resource)
	if err != nil {
		return err
	}

	response, err := l.fetcher.Get(resource)
	if err != nil {
		return err
	}

	var payload interface{}

	err = json.Unmarshal(response, &payload)
	if err != nil {
		return err
	}

	mPayload := payload.(map[string]interface{})
	responseData := mPayload[payloadItem]
	item := responseData.(map[string]interface{})

	key := itemKey
	keyParams := FindParameters(itemKey)
	for _, param := range keyParams {
		key = strings.Replace(key, param[0], item[param[1]].(string), -1)
	}

	params := make(map[string]string)
	for k, v := range itemKeyParams {
		value := v
		valueParams := FindParameters(v)
		for _, param := range valueParams {
			value = strings.Replace(value, param[0], item[param[1]].(string), -1)
		}
		params[k] = value
	}

	fetchResource, err := l.fetcher.CreateResourceFromTemplate(itemTemplate, key, params, nil)
	if err != nil {
		return err
	}

	if !l.fetcher.Exists(fetchResource.Key) {
		l.fetcher.Register(fetchResource)
	}

	err = l.fetcher.Fetch(ctx, fetchResource.Key)
	if err != nil {
		return err
	}

	return nil
}

// loadList loads a list resource
func (l *loader) loadList(ctx context.Context, resource string, payloadItems string, itemTemplate string,
	itemKey string, itemKeyParams map[string]string) error {
	err := l.fetcher.Fetch(ctx, resource)
	if err != nil {
		return err
	}

	response, err := l.fetcher.Get(resource)
	if err != nil {
		return err
	}

	var payload interface{}

	err = json.Unmarshal(response, &payload)
	if err != nil {
		return err
	}

	mPayload := payload.(map[string]interface{})
	responseData := mPayload[payloadItems]

	for _, data := range responseData.([]interface{}) {
		item := data.(map[string]interface{})

		key := itemKey
		keyParams := FindParameters(itemKey)
		for _, param := range keyParams {
			key = strings.Replace(key, param[0], item[param[1]].(string), -1)
		}

		params := make(map[string]string)
		for k, v := range itemKeyParams {
			value := v
			valueParams := FindParameters(v)
			for _, param := range valueParams {
				value = strings.Replace(value, param[0], item[param[1]].(string), -1)
			}
			params[k] = value
		}

		fetchResource, err := l.fetcher.CreateResourceFromTemplate(itemTemplate, key, params, nil)
		if err != nil {
			return err
		}

		if !l.fetcher.Exists(fetchResource.Key) {
			l.fetcher.Register(fetchResource)
		}

		err = l.fetcher.Fetch(ctx, fetchResource.Key)
		if err != nil {
			return err
		}
	}

	return nil
}
