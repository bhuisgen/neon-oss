// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"log"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/cache/memory"
	"github.com/bhuisgen/neon/pkg/core"
)

func TestStoreCheck(t *testing.T) {
	type fields struct {
		config *storeConfig
		logger *log.Logger
		state  *storeState
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
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &store{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			got, err := s.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("store.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("store.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreLoad(t *testing.T) {
	type fields struct {
		config *storeConfig
		logger *log.Logger
		state  *storeState
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
			s := &store{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("store.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStoreGet(t *testing.T) {
	data := memory.NewMemoryCache()
	data.Set("test", &core.Resource{
		Data: [][]byte{[]byte("test")},
		TTL:  0,
	}, 0)
	data.Set("invalid", "invalid", 0)

	type fields struct {
		config *storeConfig
		logger *log.Logger
		state  *storeState
	}
	type args struct {
		name string
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
				state: &storeState{
					data: data,
				},
			},
			args: args{
				name: "test",
			},
			want: &core.Resource{
				Data: [][]byte{[]byte("test")},
				TTL:  0,
			},
		},
		{
			name: "error no data",
			fields: fields{
				state: &storeState{
					data: data,
				},
			},
			args: args{
				name: "unknown",
			},
			wantErr: true,
		},
		{
			name: "error invalid data",
			fields: fields{
				state: &storeState{
					data: data,
				},
			},
			args: args{
				name: "invalid",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &store{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			got, err := s.Get(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("store.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("store.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreSet(t *testing.T) {
	type fields struct {
		config *storeConfig
		logger *log.Logger
		state  *storeState
	}
	type args struct {
		name     string
		resource *core.Resource
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
				state: &storeState{
					data: memory.NewMemoryCache(),
				},
			},
			args: args{
				name: "test",
				resource: &core.Resource{
					Data: [][]byte{[]byte("{}")},
					TTL:  0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &store{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Set(tt.args.name, tt.args.resource); (err != nil) != tt.wantErr {
				t.Errorf("store.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
