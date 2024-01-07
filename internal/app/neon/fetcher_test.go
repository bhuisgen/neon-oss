// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"log"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

func TestFetcherCheck(t *testing.T) {
	type fields struct {
		config *fetcherConfig
		logger *log.Logger
		state  *fetcherState
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
					"providers": map[string]interface{}{
						"name": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
				},
			},
		},
		{
			name: "error unregistered provider module",
			args: args{
				config: map[string]interface{}{
					"providers": map[string]interface{}{
						"name": map[string]interface{}{
							"unknown": map[string]interface{}{},
						},
					},
				},
			},
			want: []string{
				"fetcher: provider 'name', unregistered provider module 'unknown'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			got, err := f.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetcher.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetcher.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetcherLoad(t *testing.T) {
	type fields struct {
		config *fetcherConfig
		logger *log.Logger
		state  *fetcherState
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
			name: "full",
			args: args{
				config: map[string]interface{}{
					"providers": map[string]interface{}{
						"name": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
				},
			},
		},
		{
			name: "error unregistered provider module",
			args: args{
				config: map[string]interface{}{
					"providers": map[string]interface{}{
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
			f := &fetcher{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := f.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("fetcher.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetcherFetch(t *testing.T) {
	type fields struct {
		config *fetcherConfig
		logger *log.Logger
		state  *fetcherState
	}
	type args struct {
		ctx      context.Context
		name     string
		provider string
		config   map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *core.Resource
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &fetcherState{
					providers: map[string]string{
						"test": "module",
					},
					providersModules: map[string]core.FetcherProviderModule{
						"module": testFetcherProviderModule{},
					},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			want: &core.Resource{
				Data: [][]byte{[]byte("test")},
				TTL:  0,
			},
		},
		{
			name: "error resource not found",
			fields: fields{
				state: &fetcherState{},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
		{
			name: "error provider not found",
			fields: fields{
				state: &fetcherState{
					providers: map[string]string{},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
		{
			name: "error module not found",
			fields: fields{
				state: &fetcherState{
					providers: map[string]string{
						"test": "module",
					},
					providersModules: map[string]core.FetcherProviderModule{},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
		{
			name: "error fetch",
			fields: fields{
				state: &fetcherState{
					providers: map[string]string{
						"test": "module",
					},
					providersModules: map[string]core.FetcherProviderModule{
						"module": testFetcherProviderModule{
							errFetch: true,
						},
					},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			got, err := f.Fetch(tt.args.ctx, tt.args.name, tt.args.provider, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetcher.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetcher.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}
