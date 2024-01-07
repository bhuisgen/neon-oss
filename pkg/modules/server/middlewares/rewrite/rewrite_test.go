// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rewrite

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testRewriteMiddlewareServer struct {
	err bool
}

func (s testRewriteMiddlewareServer) Name() string {
	return "test"
}

func (s testRewriteMiddlewareServer) Listeners() []string {
	return nil
}

func (s testRewriteMiddlewareServer) Hosts() []string {
	return nil
}

func (s testRewriteMiddlewareServer) Store() core.Store {
	return nil
}

func (s testRewriteMiddlewareServer) Fetcher() core.Fetcher {
	return nil
}

func (s testRewriteMiddlewareServer) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testRewriteMiddlewareServer) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.Server = (*testRewriteMiddlewareServer)(nil)

func TestRewriteMiddlewareModuleInfo(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          rewriteModuleID,
				NewInstance: func() module.Module { return &rewriteMiddleware{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got := m.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("rewriteMiddleware.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("rewriteMiddleware.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestRewriteMiddlewareCheck(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
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
					"Rules": []map[string]interface{}{
						{
							"Path":        "/.*",
							"Replacement": "/test",
							"Flag":        "redirect",
						},
					},
				},
			},
		},
		{
			name: "invalid values",
			args: args{
				config: map[string]interface{}{
					"Rules": []map[string]interface{}{
						{
							"Path":        "",
							"Replacement": "",
							"Flag":        "",
						},
					},
				},
			},
			want: []string{
				"rule option 'Path', missing option or value",
				"rule option 'Replacement', missing option or value",
				"rule option 'Flag', invalid value ''",
			},
			wantErr: true,
		},
		{
			name: "invalid regular expression",
			args: args{
				config: map[string]interface{}{
					"Rules": []map[string]interface{}{
						{
							"Path":        "(",
							"Replacement": "value",
						},
					},
				},
			},
			want: []string{
				"rule option 'Path', invalid regular expression '('",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got, err := m.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("rewriteMiddleware.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rewriteMiddleware.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRewriteMiddlewareLoad(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
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
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("rewriteMiddleware.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRewriteMiddlewareRegister(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
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
				server: testRewriteMiddlewareServer{},
			},
		},
		{
			name: "error register",
			args: args{
				server: testRewriteMiddlewareServer{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Register(tt.args.server); (err != nil) != tt.wantErr {
				t.Errorf("rewriteMiddleware.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRewriteMiddlewareStart(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &rewriteMiddlewareConfig{
					Rules: []RewriteRule{
						{
							Path: "/test",
						},
					},
				},
			},
		},
		{
			name: "error regular expression",
			fields: fields{
				config: &rewriteMiddlewareConfig{
					Rules: []RewriteRule{
						{
							Path: "(",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Start(); (err != nil) != tt.wantErr {
				t.Errorf("rewriteMiddleware.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRewriteMiddlewareMount(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
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
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Mount(); (err != nil) != tt.wantErr {
				t.Errorf("rewriteMiddleware.Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRewriteMiddlewareUnmount(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
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
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			m.Unmount()
		})
	}
}

func TestRewriteMiddlewareStop(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
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
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			m.Stop()
		})
	}
}

func TestRewriteMiddlewareHandler(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
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
			m := &rewriteMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got := m.Handler(tt.args.next)
			if tt.wantNil && got != nil {
				t.Errorf("rewriteMiddleware.Handler() = %v, want %v", got, nil)
			}
		})
	}
}
