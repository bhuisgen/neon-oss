// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"log"
	"reflect"
	"testing"

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
			args: args{
				config: map[string]interface{}{
					"storage": map[string]interface{}{
						"test": map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "error no storage",
			args: args{
				config: map[string]interface{}{},
			},
			want: []string{
				"store: no storage defined",
			},
			wantErr: true,
		},
		{
			name: "error unregistered module",
			args: args{
				config: map[string]interface{}{
					"storage": map[string]interface{}{
						"unknown": map[string]interface{}{},
					},
				},
			},
			want: []string{
				"store: unregistered storage module 'unknown'",
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
			args: args{
				config: map[string]interface{}{
					"storage": map[string]interface{}{
						"test": map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "error unregistered module",
			args: args{
				config: map[string]interface{}{
					"storage": map[string]interface{}{
						"unknown": map[string]interface{}{},
					},
				},
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
			if err := s.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("store.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStoreLoadResource(t *testing.T) {
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
					storageModule: &testStoreStorageModule{},
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
			name: "error module",
			fields: fields{
				state: &storeState{
					storageModule: &testStoreStorageModule{
						errLoadResource: true,
					},
				},
			},
			args: args{
				name: "test",
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
			got, err := s.LoadResource(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("store.LoadResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("store.LoadResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreStoreResource(t *testing.T) {
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
					storageModule: &testStoreStorageModule{},
				},
			},
			args: args{
				name: "test",
				resource: &core.Resource{
					Data: [][]byte{[]byte("test")},
					TTL:  0,
				},
			},
		},
		{
			name: "error module",
			fields: fields{
				state: &storeState{
					storageModule: &testStoreStorageModule{
						errStoreResource: true,
					},
				},
			},
			args: args{
				name: "test",
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
			if err := s.StoreResource(tt.args.name, tt.args.resource); (err != nil) != tt.wantErr {
				t.Errorf("store.StoreResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
