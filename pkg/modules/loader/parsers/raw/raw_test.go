// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package raw

import (
	"context"
	"errors"
	"log"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

type testRawParserStore struct {
	errGet bool
	errSet bool
}

func (s *testRawParserStore) LoadResource(name string) (*core.Resource, error) {
	if s.errGet {
		return nil, errors.New("test error")
	}
	return nil, nil
}

func (s *testRawParserStore) StoreResource(name string, resource *core.Resource) error {
	if s.errSet {
		return errors.New("test error")
	}
	return nil
}

var _ core.Store = (*testRawParserStore)(nil)

type testRawParserFetcher struct {
	errFetch bool
}

func (f *testRawParserFetcher) Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (
	*core.Resource, error) {
	if f.errFetch {
		return nil, errors.New("test error")
	}
	return &core.Resource{}, nil
}

var _ core.Fetcher = (*testRawParserFetcher)(nil)

func TestRawParserCheck(t *testing.T) {
	type fields struct {
		config *rawParserConfig
		logger *log.Logger
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
			args: args{
				config: map[string]interface{}{
					"resource": map[string]map[string]interface{}{
						"name": {
							"provider": map[string]interface{}{},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &rawParser{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			got, err := e.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("rawParser.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rawParser.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRawParserLoad(t *testing.T) {
	type fields struct {
		config *rawParserConfig
		logger *log.Logger
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
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &rawParser{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			if err := e.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("rawParser.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRawParserParse(t *testing.T) {
	type fields struct {
		config *rawParserConfig
		logger *log.Logger
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
				config: &rawParserConfig{
					Resource: map[string]map[string]interface{}{
						"test": {
							"provider": map[string]interface{}{},
						},
					},
				},
				logger: log.Default(),
			},
			args: args{
				ctx:     context.Background(),
				store:   &testRawParserStore{},
				fetcher: &testRawParserFetcher{},
			},
		},
		{
			name: "invalid resource name",
			fields: fields{
				config: &rawParserConfig{
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
				store:   &testRawParserStore{},
				fetcher: &testRawParserFetcher{},
			},
			wantErr: true,
		},
		{
			name: "invalid provider name",
			fields: fields{
				config: &rawParserConfig{
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
				store:   &testRawParserStore{},
				fetcher: &testRawParserFetcher{},
			},
			wantErr: true,
		},
		{
			name: "error fetch",
			fields: fields{
				config: &rawParserConfig{
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
				store: &testRawParserStore{},
				fetcher: &testRawParserFetcher{
					errFetch: true,
				},
			},
			wantErr: true,
		},
		{
			name: "error store",
			fields: fields{
				config: &rawParserConfig{
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
				store: &testRawParserStore{
					errSet: true},
				fetcher: &testRawParserFetcher{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &rawParser{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			if err := e.Parse(tt.args.ctx, tt.args.store, tt.args.fetcher); (err != nil) != tt.wantErr {
				t.Errorf("rawParser.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
