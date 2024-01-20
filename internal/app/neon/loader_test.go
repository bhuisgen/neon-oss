// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"log"
	"testing"
)

func intPtr(i int) *int {
	return &i
}

func TestLoaderInit(t *testing.T) {
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
			name: "no configuration",
			fields: fields{
				logger: log.Default(),
			},
		},
		{
			name: "empty configuration",
			fields: fields{
				logger: log.Default(),
			},
			args: args{
				config: map[string]interface{}{},
			},
		},
		{
			name: "full",
			fields: fields{
				logger: log.Default(),
			},
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
			fields: fields{
				logger: log.Default(),
			},
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
			wantErr: true,
		},
		{
			name: "error unregistered parser module",
			fields: fields{
				logger: log.Default(),
			},
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
			if err := l.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("loader.Init() error = %v, wantErr %v", err, tt.wantErr)
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
			name: "execution",
			fields: fields{
				config: &loaderConfig{
					ExecInterval: intPtr(loaderConfigDefaultExecInterval),
					ExecStartup:  intPtr(loaderConfigDefaultExecStartup),
				},
				logger: log.Default(),
				state:  &loaderState{},
				stop:   make(chan struct{}, 1),
			},
		},
		{
			name: "no execution",
			fields: fields{
				config: &loaderConfig{
					ExecInterval: intPtr(0),
					ExecStartup:  intPtr(loaderConfigDefaultExecStartup),
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
			name: "execution",
			fields: fields{
				config: &loaderConfig{
					ExecInterval: intPtr(loaderConfigDefaultExecInterval),
				},
				logger: log.Default(),
				state:  &loaderState{},
				stop:   make(chan struct{}, 1),
			},
		},
		{
			name: "no execution",
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
