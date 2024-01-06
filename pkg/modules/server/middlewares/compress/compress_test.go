// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package compress

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testCompressMiddlewareServer struct {
	err bool
}

func (s testCompressMiddlewareServer) Name() string {
	return "test"
}

func (s testCompressMiddlewareServer) Listeners() []string {
	return nil
}

func (s testCompressMiddlewareServer) Hosts() []string {
	return nil
}

func (s testCompressMiddlewareServer) Store() core.Store {
	return nil
}

func (s testCompressMiddlewareServer) Fetcher() core.Fetcher {
	return nil
}

func (s testCompressMiddlewareServer) RegisterMiddleware(
	middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testCompressMiddlewareServer) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.Server = (*testCompressMiddlewareServer)(nil)

func TestCompressMiddlewareModuleInfo(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          compressModuleID,
				NewInstance: func() module.Module { return &compressMiddleware{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			got := m.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("compressMiddleware.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("compressMiddleware.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestCompressMiddlewareCheck(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
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
			args: args{},
		},
		{
			name: "full",
			args: args{
				config: map[string]interface{}{
					"Level": -1,
				},
			},
		},
		{
			name: "invalid config",
			args: args{
				config: map[string]interface{}{
					"Level": 100,
				},
			},
			want: []string{
				"option 'Level', invalid value '100'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			got, err := m.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("compressMiddleware.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompressMiddlewareLoad(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
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
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareRegister(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
	}
	type args struct {
		server core.Server
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
				server: testCompressMiddlewareServer{},
			},
		},
		{
			name: "error register",
			args: args{
				server: testCompressMiddlewareServer{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Register(tt.args.server); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareStart(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Start(); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareMount(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			if err := m.Mount(); (err != nil) != tt.wantErr {
				t.Errorf("compressMiddleware.Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressMiddlewareUnmount(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			m.Unmount()
		})
	}
}

func TestCompressMiddlewareStop(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			m.Stop()
		})
	}
}

func TestCompressMiddlewareHandler(t *testing.T) {
	type fields struct {
		config *compressMiddlewareConfig
		logger *log.Logger
		pool   *gzipPool
	}
	type args struct {
		next http.Handler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &compressMiddleware{
				config: tt.fields.config,
				logger: tt.fields.logger,
				pool:   tt.fields.pool,
			}
			got := m.Handler(tt.args.next)
			if tt.wantNil && got != nil {
				t.Errorf("compressMiddleware.Handler() = %v, want %v", got, nil)
			}
		})
	}
}
