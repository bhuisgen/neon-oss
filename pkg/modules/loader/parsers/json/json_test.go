// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package json

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

type testJSONParserStore struct {
	errGet bool
	errSet bool
}

func (s *testJSONParserStore) LoadResource(name string) (*core.Resource, error) {
	if s.errGet {
		return nil, errors.New("test error")
	}
	return nil, nil
}

func (s *testJSONParserStore) StoreResource(name string, resource *core.Resource) error {
	if s.errSet {
		return errors.New("test error")
	}
	return nil
}

var _ core.Store = (*testJSONParserStore)(nil)

type testJSONParserFetcher struct {
	resource *core.Resource
	errFetch bool
}

func (f *testJSONParserFetcher) Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (
	*core.Resource, error) {
	if f.errFetch {
		return nil, errors.New("test error")
	}
	return f.resource, nil
}

var _ core.Fetcher = (*testJSONParserFetcher)(nil)

func TestJSONParserInit(t *testing.T) {
	type fields struct {
		config        *jsonParserConfig
		logger        *log.Logger
		jsonUnmarshal func(data []byte, v any) error
	}
	type args struct {
		config map[string]interface{}
		logger *log.Logger
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
			args: args{
				config: map[string]interface{}{
					"resource": map[string]map[string]interface{}{
						"name": {
							"provider": map[string]interface{}{},
						},
					},
					"filter": "$.results",
					"itemResource": map[string]map[string]interface{}{
						"resource": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: log.Default(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &jsonParser{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				jsonUnmarshal: tt.fields.jsonUnmarshal,
			}
			if err := p.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("restProvider.Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestJSONParserParse(t *testing.T) {
	type fields struct {
		config        *jsonParserConfig
		logger        *log.Logger
		jsonUnmarshal func(data []byte, v any) error
	}
	type args struct {
		ctx     context.Context
		store   core.Store
		fetcher core.Fetcher
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
				config: &jsonParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
					Filter: "$.results",
					ItemResource: map[string]map[string]interface{}{
						"resource": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger:        log.Default(),
				jsonUnmarshal: json.Unmarshal,
			},
			args: args{
				ctx:   context.Background(),
				store: &testJSONParserStore{},
				fetcher: &testJSONParserFetcher{
					resource: &core.Resource{
						Data: [][]byte{},
						TTL:  0,
					},
				},
			},
		},
		{
			name: "invalid resource name",
			fields: fields{
				config: &jsonParserConfig{
					Resource: map[string]map[string]interface{}{
						"": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: log.Default(),
			},
			args: args{
				ctx:     context.Background(),
				store:   &testJSONParserStore{},
				fetcher: &testJSONParserFetcher{},
			},
			wantErr: true,
		},
		{
			name: "invalid provider name",
			fields: fields{
				config: &jsonParserConfig{
					Resource: map[string]map[string]interface{}{
						"name": {
							"": map[string]interface{}{},
						},
					},
				},
				logger: log.Default(),
			},
			args: args{
				ctx:     context.Background(),
				store:   &testJSONParserStore{},
				fetcher: &testJSONParserFetcher{},
			},
			wantErr: true,
		},
		{
			name: "error fetch",
			fields: fields{
				config: &jsonParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: log.Default(),
			},
			args: args{
				ctx:   context.Background(),
				store: &testJSONParserStore{},
				fetcher: &testJSONParserFetcher{
					errFetch: true,
				},
			},
			wantErr: true,
		},
		{
			name: "error store",
			fields: fields{
				config: &jsonParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: log.Default(),
			},
			args: args{
				ctx: context.Background(),
				store: &testJSONParserStore{
					errSet: true},
				fetcher: &testJSONParserFetcher{
					resource: &core.Resource{
						Data: [][]byte{},
						TTL:  0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "fetch resource",
			fields: fields{
				config: &jsonParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
					Filter: "$.results",
					ItemResource: map[string]map[string]interface{}{
						"resource": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger:        log.Default(),
				jsonUnmarshal: json.Unmarshal,
			},
			args: args{
				ctx:   context.Background(),
				store: &testJSONParserStore{},
				fetcher: &testJSONParserFetcher{
					resource: &core.Resource{
						Data: [][]byte{[]byte("{\"results\":[{\"name\":\"one\"}]}")},
						TTL:  0,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &jsonParser{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				jsonUnmarshal: tt.fields.jsonUnmarshal,
			}
			if err := p.Parse(tt.args.ctx, tt.args.store, tt.args.fetcher); (err != nil) != tt.wantErr {
				t.Errorf("jsonParser.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
