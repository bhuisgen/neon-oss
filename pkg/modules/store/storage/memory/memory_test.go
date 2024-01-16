// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"errors"
	"log"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testMemoryStore struct {
	errLoadResource  bool
	errStoreResource bool
}

// LoadResource implements core.Store.
func (s testMemoryStore) LoadResource(name string) (*core.Resource, error) {
	if s.errLoadResource {
		return nil, errors.New("test error")
	}
	return &core.Resource{
		Data: [][]byte{[]byte("test")},
		TTL:  0,
	}, nil
}

// StoreResource implements core.Store.
func (s testMemoryStore) StoreResource(name string, resource *core.Resource) error {
	if s.errStoreResource {
		return errors.New("test error")
	}
	return nil
}

var _ core.Store = (*testMemoryStore)(nil)

type testMemoryCache struct {
	errGet bool
}

func (c testMemoryCache) Get(key string) any {
	if c.errGet {
		return nil
	}

	return &core.Resource{
		Data: [][]byte{[]byte("test")},
		TTL:  0,
	}
}

func (c testMemoryCache) Set(key string, value any) {
}

func (c testMemoryCache) Remove(key string) {
}

func (c testMemoryCache) Clear() {
}

var _ cache.Cache = (*testMemoryCache)(nil)

func TestMemoryStorageModuleInfo(t *testing.T) {
	type fields struct {
		config  *memoryStorageConfig
		logger  *log.Logger
		storage cache.Cache
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          memoryModuleID,
				NewInstance: func() module.Module { return &memoryStorage{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := memoryStorage{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			got := s.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("memoryStorage.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("memoryStorage.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestMemoryStorageCheck(t *testing.T) {
	type fields struct {
		config  *memoryStorageConfig
		logger  *log.Logger
		storage cache.Cache
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
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &memoryStorage{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			got, err := s.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("memoryStorage.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("memoryStorage.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryStorageLoad(t *testing.T) {
	type fields struct {
		config  *memoryStorageConfig
		logger  *log.Logger
		storage cache.Cache
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
			s := &memoryStorage{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			if err := s.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("memoryStorage.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryStorageLoadResource(t *testing.T) {
	type fields struct {
		config  *memoryStorageConfig
		logger  *log.Logger
		storage cache.Cache
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
				config:  &memoryStorageConfig{},
				storage: testMemoryCache{},
			},
			args: args{
				name: "test",
			},
		},
		{
			name: "invalid resource name",
			fields: fields{
				config: &memoryStorageConfig{},
				storage: testMemoryCache{
					errGet: true,
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
			s := &memoryStorage{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			_, err := s.LoadResource(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("memoryStorage.LoadResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestMemoryStorageStoreResource(t *testing.T) {
	type fields struct {
		config  *memoryStorageConfig
		logger  *log.Logger
		storage cache.Cache
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
				config:  &memoryStorageConfig{},
				storage: testMemoryCache{},
			},
			args: args{
				name: "test",
				resource: &core.Resource{
					Data: [][]byte{[]byte("test")},
					TTL:  0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &memoryStorage{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				storage: tt.fields.storage,
			}
			if err := s.StoreResource(tt.args.name, tt.args.resource); (err != nil) != tt.wantErr {
				t.Errorf("memoryStorage.StoreResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
