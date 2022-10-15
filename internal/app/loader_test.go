// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"testing"
)

type testLoaderExecutor struct{}

func (e testLoaderExecutor) execute(stop <-chan struct{}) {
}

type testLoaderExecutorFetcher struct {
	errFetch                           bool
	errFetchOnlyForNames               []string
	exists                             bool
	get                                []byte
	errGet                             bool
	createResourceFromTemplateResource *Resource
	errCreateResourceFromTemplate      bool
}

func (t testLoaderExecutorFetcher) Fetch(ctx context.Context, name string) error {
	if t.errFetch {
		return errors.New("test error")
	}
	for _, n := range t.errFetchOnlyForNames {
		if n == name {
			return errors.New("test error")
		}
	}
	return nil
}

func (t testLoaderExecutorFetcher) Exists(name string) bool {
	return t.exists
}

func (t testLoaderExecutorFetcher) Get(name string) ([]byte, error) {
	if t.errGet {
		return nil, errors.New("test error")
	}
	return t.get, nil
}

func (t testLoaderExecutorFetcher) Register(r *Resource) {
}

func (t testLoaderExecutorFetcher) Unregister(name string) {
}

func (t testLoaderExecutorFetcher) CreateResourceFromTemplate(template string, resource string, params map[string]string,
	headers map[string]string) (*Resource, error) {
	if t.errCreateResourceFromTemplate {
		return nil, errors.New("test error")
	}
	return t.createResourceFromTemplateResource, nil
}

func TestCreateLoader(t *testing.T) {
	type args struct {
		config  *LoaderConfig
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config:  &LoaderConfig{},
				fetcher: &fetcher{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateLoader(tt.args.config, tt.args.fetcher)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateLoader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestLoaderStart(t *testing.T) {
	type fields struct {
		config   *LoaderConfig
		logger   *log.Logger
		executor *testLoaderExecutor
		stop     chan struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &LoaderConfig{
					ExecInterval: configDefaultLoaderExecInterval,
				},
				logger:   log.Default(),
				executor: &testLoaderExecutor{},
				stop:     make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &loader{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				executor: tt.fields.executor,
				stop:     tt.fields.stop,
			}
			if err := l.Start(); (err != nil) != tt.wantErr {
				t.Errorf("loader.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoaderStop(t *testing.T) {
	type fields struct {
		config   *LoaderConfig
		logger   *log.Logger
		executor *testLoaderExecutor
		stop     chan struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &LoaderConfig{
					ExecInterval: configDefaultLoaderExecInterval,
				},
				logger:   log.Default(),
				executor: &testLoaderExecutor{},
				stop:     make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &loader{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				executor: tt.fields.executor,
				stop:     tt.fields.stop,
			}
			if err := l.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("loader.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoaderExecutorExecute(t *testing.T) {
	type fields struct {
		config        *LoaderConfig
		logger        *log.Logger
		failsafe      bool
		fetcher       Fetcher
		jsonUnmarshal func(data []byte, v any) error
	}
	type args struct {
		stop <-chan struct{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				config: &LoaderConfig{
					ExecStartup:  1,
					ExecInterval: 1,
					ExecWorkers:  1,
				},
				logger: log.Default(),
			},
			args: args{
				stop: make(<-chan struct{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := loaderExecutor{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				fetcher:       tt.fields.fetcher,
				jsonUnmarshal: tt.fields.jsonUnmarshal,
			}
			e.execute(tt.args.stop)
		})
	}
}

func TestLoaderExecutorLoadStatic(t *testing.T) {
	type fields struct {
		config   *LoaderConfig
		logger   *log.Logger
		failsafe bool
		fetcher  Fetcher
	}
	type args struct {
		ctx  context.Context
		rule *LoaderRuleStatic
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config:  &LoaderConfig{},
				logger:  log.Default(),
				fetcher: testLoaderExecutorFetcher{},
			},
			args: args{
				ctx:  context.Background(),
				rule: &LoaderRuleStatic{},
			},
		},
		{
			name: "error",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					errFetch: true,
				},
			},
			args: args{
				ctx:  context.Background(),
				rule: &LoaderRuleStatic{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := loaderExecutor{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				fetcher:       tt.fields.fetcher,
				jsonUnmarshal: loaderJsonUnmarshal,
			}
			if err := e.loadStatic(tt.args.ctx, tt.args.rule); (err != nil) != tt.wantErr {
				t.Errorf("loaderExecutor.loadStatic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoaderExecutorLoadSingle(t *testing.T) {
	type fields struct {
		config        *LoaderConfig
		logger        *log.Logger
		failsafe      bool
		fetcher       Fetcher
		jsonUnmarshal func(data []byte, v any) error
	}
	type args struct {
		ctx  context.Context
		rule *LoaderRuleSingle
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					exists: false,
					get:    []byte(`{"data": {"id": 1, "string": "test", "float": -1.00, "bool": true}}`),
					createResourceFromTemplateResource: &Resource{
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
					},
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleSingle{
					Resource:             "resource",
					ResourcePayloadItem:  "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
		},
		{
			name: "error fetcher fetch",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					errFetch: true,
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleSingle{
					Resource:             "resource",
					ResourcePayloadItem:  "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error fetcher get",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					errGet: true,
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleSingle{
					Resource:             "resource",
					ResourcePayloadItem:  "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error json unmarshal",
			fields: fields{
				config:  &LoaderConfig{},
				logger:  log.Default(),
				fetcher: testLoaderExecutorFetcher{},
				jsonUnmarshal: func(data []byte, v any) error {
					return errors.New("test error")
				},
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleSingle{
					Resource:             "resource",
					ResourcePayloadItem:  "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error fetcher create resource from template",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					exists:                        false,
					get:                           []byte(`{"data": {"id": 1, "slug": "test"}}`),
					errCreateResourceFromTemplate: true,
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleSingle{
					Resource:             "resource",
					ResourcePayloadItem:  "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error fetcher fetch new resource",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					exists: false,
					get:    []byte(`{"data": {"id": 1, "slug": "test"}}`),
					createResourceFromTemplateResource: &Resource{
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
					},
					errFetchOnlyForNames: []string{"test"},
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleSingle{
					Resource:             "resource",
					ResourcePayloadItem:  "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := loaderExecutor{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				fetcher:       tt.fields.fetcher,
				jsonUnmarshal: tt.fields.jsonUnmarshal,
			}
			if err := e.loadSingle(tt.args.ctx, tt.args.rule); (err != nil) != tt.wantErr {
				t.Errorf("loaderExecutor.loadSingle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoaderExecutorLoadList(t *testing.T) {
	type fields struct {
		config        *LoaderConfig
		logger        *log.Logger
		failsafe      bool
		fetcher       Fetcher
		jsonUnmarshal func(data []byte, v any) error
	}
	type args struct {
		ctx  context.Context
		rule *LoaderRuleList
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					exists: false,
					get:    []byte(`{"data": [{"id": 1, "string": "test1", "float": -1.00, "bool": true}]}`),
					createResourceFromTemplateResource: &Resource{
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
					},
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleList{
					Resource:             "resource",
					ResourcePayloadItems: "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
		},
		{
			name: "error fetcher fetch",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					errFetch: true,
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleList{
					Resource:             "resource",
					ResourcePayloadItems: "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error fetcher get",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					errGet: true,
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleList{
					Resource:             "resource",
					ResourcePayloadItems: "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error json unmarshal",
			fields: fields{
				config:  &LoaderConfig{},
				logger:  log.Default(),
				fetcher: testLoaderExecutorFetcher{},
				jsonUnmarshal: func(data []byte, v any) error {
					return errors.New("test error")
				},
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleList{
					Resource:             "resource",
					ResourcePayloadItems: "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error fetcher create resource from template",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					exists:                        false,
					get:                           []byte(`{"data": [{"id": 1, "string": "test1", "float": -1.00, "bool": true}]}`),
					errCreateResourceFromTemplate: true,
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleList{
					Resource:             "resource",
					ResourcePayloadItems: "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error fetcher fetch new resource",
			fields: fields{
				config: &LoaderConfig{},
				logger: log.Default(),
				fetcher: testLoaderExecutorFetcher{
					exists: false,
					get:    []byte(`{"data": [{"id": 1, "string": "test1", "float": -1.00, "bool": true}]}`),
					createResourceFromTemplateResource: &Resource{
						Name:   "test",
						Method: http.MethodGet,
						URL:    "http://localhost",
					},
					errFetchOnlyForNames: []string{"test"},
				},
				jsonUnmarshal: loaderJsonUnmarshal,
			},
			args: args{
				ctx: context.Background(),
				rule: &LoaderRuleList{
					Resource:             "resource",
					ResourcePayloadItems: "data",
					ItemTemplate:         "template",
					ItemTemplateResource: "resource-$id",
					ItemTemplateResourceParams: map[string]string{
						"id":   "$id",
						"slug": "$slug",
					},
					ItemTemplateResourceHeaders: map[string]string{
						"header1": "value1",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := loaderExecutor{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				fetcher:       tt.fields.fetcher,
				jsonUnmarshal: tt.fields.jsonUnmarshal,
			}
			if err := e.loadList(tt.args.ctx, tt.args.rule); (err != nil) != tt.wantErr {
				t.Errorf("loaderExecutor.loadList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
