// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rewrite

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

type testRewriteMiddlewareServerSite struct {
	err bool
}

func (s testRewriteMiddlewareServerSite) Name() string {
	return "test"
}

func (s testRewriteMiddlewareServerSite) Listeners() []string {
	return nil
}

func (s testRewriteMiddlewareServerSite) Hosts() []string {
	return nil
}

func (s testRewriteMiddlewareServerSite) Store() core.Store {
	return nil
}

func (s testRewriteMiddlewareServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testRewriteMiddlewareServerSite) Loader() core.Loader {
	return nil
}

func (s testRewriteMiddlewareServerSite) Server() core.Server {
	return nil
}

func (s testRewriteMiddlewareServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testRewriteMiddlewareServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testRewriteMiddlewareServerSite)(nil)

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

func TestRewriteMiddlewareInit(t *testing.T) {
	type fields struct {
		config  *rewriteMiddlewareConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
	}
	type args struct {
		config map[string]interface{}
		logger *log.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "minimal",
			args: args{
				config: map[string]interface{}{},
				logger: log.Default(),
			},
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
				logger: log.Default(),
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
				logger: log.Default(),
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
				logger: log.Default(),
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
			if err := m.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("rewriteMiddleware.Init() error = %v, wantErr %v", err, tt.wantErr)
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
		site core.ServerSite
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
				site: testRewriteMiddlewareServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testRewriteMiddlewareServerSite{
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
			if err := m.Register(tt.args.site); (err != nil) != tt.wantErr {
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
