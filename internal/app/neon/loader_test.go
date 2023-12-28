// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"log"
	"reflect"
	"testing"
)

func intPtr(i int) *int {
	return &i
}

func TestLoaderCheck(t *testing.T) {
	type fields struct {
		config  *loaderConfig
		logger  *log.Logger
		state   *loaderState
		fetcher Fetcher
		stop    chan struct{}
	}
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "minimal",
		},
		{
			name: "full",
			args: args{
				config: map[string]interface{}{
					"execStartup":          15,
					"execInterval":         60,
					"execFailsafeInterval": 15,
					"execWorkers":          1,
					"execMaxOps":           100,
					"execMaxDelay":         1,
					"rules": map[string]interface{}{
						"test": map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "invalid values",
			args: args{
				config: map[string]interface{}{
					"execStartup":          -1,
					"execInterval":         -1,
					"execFailsafeInterval": -1,
					"execWorkers":          -1,
					"execMaxOps":           -1,
					"execMaxDelay":         -1,
				},
			},
			want: []string{
				"loader: option 'ExecStartup', invalid value '-1'",
				"loader: option 'ExecInterval', invalid value '-1'",
				"loader: option 'ExecFailsafeInterval', invalid value '-1'",
				"loader: option 'ExecWorkers', invalid value '-1'",
				"loader: option 'ExecMaxOps', invalid value '-1'",
				"loader: option 'ExecMaxDelay', invalid value '-1'",
			},
			wantErr: true,
		},
		{
			name: "error unregistered parser module",
			args: args{
				config: map[string]interface{}{
					"rules": map[string]interface{}{
						"name": map[string]interface{}{
							"unknown": map[string]interface{}{},
						},
					},
				},
			},
			want: []string{
				"loader: rule 'name', unregistered parser module 'unknown'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &loader{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
				stop:    tt.fields.stop,
			}
			got, err := l.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("loader.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loader.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoaderLoad(t *testing.T) {
	type fields struct {
		config  *loaderConfig
		logger  *log.Logger
		state   *loaderState
		fetcher Fetcher
		stop    chan struct{}
	}
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "minimal",
		},
		{
			name: "error unregistered parser module",
			args: args{
				config: map[string]interface{}{
					"rules": map[string]interface{}{
						"name": map[string]interface{}{
							"unknown": map[string]interface{}{},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &loader{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
				stop:    tt.fields.stop,
			}
			if err := l.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("loader.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoaderStart(t *testing.T) {
	type fields struct {
		config  *loaderConfig
		logger  *log.Logger
		state   *loaderState
		fetcher Fetcher
		stop    chan struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &loaderConfig{
					ExecInterval: intPtr(0),
				},
				logger: log.Default(),
				state:  &loaderState{},
				stop:   make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &loader{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
				stop:    tt.fields.stop,
			}
			if err := l.Start(); (err != nil) != tt.wantErr {
				t.Errorf("loader.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoaderStop(t *testing.T) {
	type fields struct {
		config  *loaderConfig
		logger  *log.Logger
		state   *loaderState
		fetcher Fetcher
		stop    chan struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &loaderConfig{
					ExecInterval: intPtr(0),
				},
				logger: log.Default(),
				state:  &loaderState{},
				stop:   make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &loader{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
				stop:    tt.fields.stop,
			}
			if err := l.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("loader.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// func TestLoaderExecutorExecute(t *testing.T) {
// 	type fields struct {
// 		config        *loaderConfig
// 		logger        *log.Logger
// 		fetcher       Fetcher
// 		failsafe      bool
// 		jsonUnmarshal func(data []byte, v any) error
// 	}
// 	type args struct {
// 		stop <-chan struct{}
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		args   args
// 	}{
// 		{
// 			name: "default",
// 			fields: fields{
// 				config: &loaderConfig{
// 					ExecStartup:  intPtr(1),
// 					ExecInterval: intPtr(1),
// 					ExecWorkers:  intPtr(1),
// 				},
// 				logger: log.Default(),
// 			},
// 			args: args{
// 				stop: make(<-chan struct{}),
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			e := loaderExecutor{
// 				config:        tt.fields.config,
// 				logger:        tt.fields.logger,
// 				fetcher:       tt.fields.fetcher,
// 				failsafe:      tt.fields.failsafe,
// 				jsonUnmarshal: tt.fields.jsonUnmarshal,
// 			}
// 			e.execute(tt.args.stop)
// 		})
// 	}
// }

// // func TestLoaderExecutorLoadStatic(t *testing.T) {
// // 	type fields struct {
// // 		config        *loaderConfig
// // 		logger        *log.Logger
// // 		fetcher       Fetcher
// // 		failsafe      bool
// // 		jsonUnmarshal func(data []byte, v any) error
// // 	}
// // 	type args struct {
// // 		ctx  context.Context
// // 		rule *loaderRuleStatic
// // 	}
// // 	tests := []struct {
// // 		name    string
// // 		fields  fields
// // 		args    args
// // 		wantErr bool
// // 	}{
// // 		{
// // 			name: "default",
// // 			fields: fields{
// // 				config:  &loaderConfig{},
// // 				logger:  log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{},
// // 			},
// // 			args: args{
// // 				ctx:  context.Background(),
// // 				rule: &loaderRuleStatic{},
// // 			},
// // 		},
// // 		{
// // 			name: "error",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					errFetch: true,
// // 				},
// // 			},
// // 			args: args{
// // 				ctx:  context.Background(),
// // 				rule: &loaderRuleStatic{},
// // 			},
// // 			wantErr: true,
// // 		},
// // 	}
// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			e := loaderExecutor{
// // 				config:        tt.fields.config,
// // 				logger:        tt.fields.logger,
// // 				fetcher:       tt.fields.fetcher,
// // 				failsafe:      tt.fields.failsafe,
// // 				jsonUnmarshal: tt.fields.jsonUnmarshal,
// // 			}
// // 			if err := e.loadStatic(tt.args.ctx, tt.args.rule); (err != nil) != tt.wantErr {
// // 				t.Errorf("loaderExecutor.loadStatic() error = %v, wantErr %v", err, tt.wantErr)
// // 			}
// // 		})
// // 	}
// // }

// // func TestLoaderExecutorLoadSingle(t *testing.T) {
// // 	type fields struct {
// // 		config        *loaderConfig
// // 		logger        *log.Logger
// // 		fetcher       Fetcher
// // 		failsafe      bool
// // 		jsonUnmarshal func(data []byte, v any) error
// // 	}
// // 	type args struct {
// // 		ctx  context.Context
// // 		rule *loaderRuleSingle
// // 	}
// // 	tests := []struct {
// // 		name    string
// // 		fields  fields
// // 		args    args
// // 		wantErr bool
// // 	}{
// // 		{
// // 			name: "default",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					exists: false,
// // 					get:    []byte(`{"data": {"id": 1, "string": "test", "float": -1.00, "bool": true}}`),
// // 					createResourceFromTemplateResource: &core.Resource{
// // 						Name:   "test",
// // 						Method: http.MethodGet,
// // 						URL:    "http://localhost",
// // 					},
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleSingle{
// // 					Resource:             "resource",
// // 					ResourcePayloadItem:  "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 		},
// // 		{
// // 			name: "error fetcher fetch",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					errFetch: true,
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleSingle{
// // 					Resource:             "resource",
// // 					ResourcePayloadItem:  "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error fetcher get",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					errGet: true,
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleSingle{
// // 					Resource:             "resource",
// // 					ResourcePayloadItem:  "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error json unmarshal",
// // 			fields: fields{
// // 				config:  &loaderConfig{},
// // 				logger:  log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{},
// // 				jsonUnmarshal: func(data []byte, v any) error {
// // 					return errors.New("test error")
// // 				},
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleSingle{
// // 					Resource:             "resource",
// // 					ResourcePayloadItem:  "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error fetcher create resource from template",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					exists:                        false,
// // 					get:                           []byte(`{"data": {"id": 1, "slug": "test"}}`),
// // 					errCreateResourceFromTemplate: true,
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleSingle{
// // 					Resource:             "resource",
// // 					ResourcePayloadItem:  "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error fetcher fetch new resource",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					exists: false,
// // 					get:    []byte(`{"data": {"id": 1, "slug": "test"}}`),
// // 					createResourceFromTemplateResource: &core.Resource{
// // 						Name:   "test",
// // 						Method: http.MethodGet,
// // 						URL:    "http://localhost",
// // 					},
// // 					errFetchOnlyForNames: []string{"test"},
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleSingle{
// // 					Resource:             "resource",
// // 					ResourcePayloadItem:  "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 	}
// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			e := loaderExecutor{
// // 				config:        tt.fields.config,
// // 				logger:        tt.fields.logger,
// // 				fetcher:       tt.fields.fetcher,
// // 				failsafe:      tt.fields.failsafe,
// // 				jsonUnmarshal: tt.fields.jsonUnmarshal,
// // 			}
// // 			if err := e.loadSingle(tt.args.ctx, tt.args.rule); (err != nil) != tt.wantErr {
// // 				t.Errorf("loaderExecutor.loadSingle() error = %v, wantErr %v", err, tt.wantErr)
// // 			}
// // 		})
// // 	}
// // }

// // func TestLoaderExecutorLoadList(t *testing.T) {
// // 	type fields struct {
// // 		config        *loaderConfig
// // 		logger        *log.Logger
// // 		fetcher       Fetcher
// // 		failsafe      bool
// // 		jsonUnmarshal func(data []byte, v any) error
// // 	}
// // 	type args struct {
// // 		ctx  context.Context
// // 		rule *loaderRuleList
// // 	}
// // 	tests := []struct {
// // 		name    string
// // 		fields  fields
// // 		args    args
// // 		wantErr bool
// // 	}{
// // 		{
// // 			name: "default",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					exists: false,
// // 					get:    []byte(`{"data": [{"id": 1, "string": "test1", "float": -1.00, "bool": true}]}`),
// // 					createResourceFromTemplateResource: &core.Resource{
// // 						Name:   "test",
// // 						Method: http.MethodGet,
// // 						URL:    "http://localhost",
// // 					},
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleList{
// // 					Resource:             "resource",
// // 					ResourcePayloadItems: "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 		},
// // 		{
// // 			name: "error fetcher fetch",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					errFetch: true,
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleList{
// // 					Resource:             "resource",
// // 					ResourcePayloadItems: "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error fetcher get",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					errGet: true,
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleList{
// // 					Resource:             "resource",
// // 					ResourcePayloadItems: "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error json unmarshal",
// // 			fields: fields{
// // 				config:  &loaderConfig{},
// // 				logger:  log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{},
// // 				jsonUnmarshal: func(data []byte, v any) error {
// // 					return errors.New("test error")
// // 				},
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleList{
// // 					Resource:             "resource",
// // 					ResourcePayloadItems: "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error fetcher create resource from template",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					exists: false,
// // 					get: []byte(
// // 						`{"data": [{"id": 1, "string": "test1", "float": -1.00, "bool": true}]}`),
// // 					errCreateResourceFromTemplate: true,
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleList{
// // 					Resource:             "resource",
// // 					ResourcePayloadItems: "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 		{
// // 			name: "error fetcher fetch new resource",
// // 			fields: fields{
// // 				config: &loaderConfig{},
// // 				logger: log.Default(),
// // 				fetcher: testLoaderExecutorFetcher{
// // 					exists: false,
// // 					get:    []byte(`{"data": [{"id": 1, "string": "test1", "float": -1.00, "bool": true}]}`),
// // 					createResourceFromTemplateResource: &core.Resource{
// // 						Name:   "test",
// // 						Method: http.MethodGet,
// // 						URL:    "http://localhost",
// // 					},
// // 					errFetchOnlyForNames: []string{"test"},
// // 				},
// // 				jsonUnmarshal: loaderJsonUnmarshal,
// // 			},
// // 			args: args{
// // 				ctx: context.Background(),
// // 				rule: &loaderRuleList{
// // 					Resource:             "resource",
// // 					ResourcePayloadItems: "data",
// // 					ItemTemplate:         "template",
// // 					ItemTemplateResource: "resource-$id",
// // 					ItemTemplateResourceParams: map[string]string{
// // 						"id":   "$id",
// // 						"slug": "$slug",
// // 					},
// // 					ItemTemplateResourceHeaders: map[string]string{
// // 						"header1": "value1",
// // 					},
// // 				},
// // 			},
// // 			wantErr: true,
// // 		},
// // 	}
// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			e := loaderExecutor{
// // 				config:        tt.fields.config,
// // 				logger:        tt.fields.logger,
// // 				fetcher:       tt.fields.fetcher,
// // 				failsafe:      tt.fields.failsafe,
// // 				jsonUnmarshal: tt.fields.jsonUnmarshal,
// // 			}
// // 			if err := e.loadList(tt.args.ctx, tt.args.rule); (err != nil) != tt.wantErr {
// // 				t.Errorf("loaderExecutor.loadList() error = %v, wantErr %v", err, tt.wantErr)
// // 			}
// // 		})
// // 	}
// // }

// // func TestReplaceLoaderResourceParameters(t *testing.T) {
// // 	type args struct {
// // 		s      string
// // 		params map[string]string
// // 	}
// // 	tests := []struct {
// // 		name string
// // 		args args
// // 		want string
// // 	}{
// // 		{
// // 			name: "default",
// // 			args: args{
// // 				s: "test-$value1-$value2-$value3",
// // 				params: map[string]string{
// // 					"value1": "one",
// // 					"value2": "two",
// // 					"value3": "three",
// // 				},
// // 			},
// // 			want: "test-one-two-three",
// // 		},
// // 	}
// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			if got := replaceLoaderResourceParameters(tt.args.s, tt.args.params); got != tt.want {
// // 				t.Errorf("replaceLoaderResourceParameters() = %v, want %v", got, tt.want)
// // 			}
// // 		})
// // 	}
// // }
