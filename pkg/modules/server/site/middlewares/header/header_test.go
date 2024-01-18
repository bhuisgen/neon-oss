// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package header

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

type testHeaderMiddlewareServerSite struct {
	err bool
}

func (s testHeaderMiddlewareServerSite) Name() string {
	return "test"
}

func (s testHeaderMiddlewareServerSite) Listeners() []string {
	return nil
}

func (s testHeaderMiddlewareServerSite) Hosts() []string {
	return nil
}

func (s testHeaderMiddlewareServerSite) Store() core.Store {
	return nil
}

func (s testHeaderMiddlewareServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testHeaderMiddlewareServerSite) Loader() core.Loader {
	return nil
}

func (s testHeaderMiddlewareServerSite) Server() core.Server {
	return nil
}

func (s testHeaderMiddlewareServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testHeaderMiddlewareServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testHeaderMiddlewareServerSite)(nil)

func TestHeaderMiddlewareModuleInfo(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
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
				ID:          headerModuleID,
				NewInstance: func() module.Module { return &headerMiddleware{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got := m.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("headerMiddleware.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("headerMiddleware.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestHeaderMiddlewareCheck(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
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
							"Path": "/.*",
							"Set": map[string]string{
								"test1": "value1",
							},
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
							"Path": "",
							"Set": map[string]string{
								"": "",
							},
							"Remove": []string{""},
						},
					},
				},
			},
			want: []string{
				"rule option 'Path', missing option or value",
				"rule option 'Set', invalid key ''",
			},
			wantErr: true,
		},
		{
			name: "invalid regular expression",
			args: args{
				config: map[string]interface{}{
					"Rules": []map[string]interface{}{
						{
							"Path": "(",
							"Set": map[string]string{
								"test": "value",
							},
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got, err := m.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("headerMiddleware.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderMiddlewareLoad(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderMiddlewareRegister(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
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
				site: testHeaderMiddlewareServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testHeaderMiddlewareServerSite{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Register(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderMiddlewareStart(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
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
				config: &headerMiddlewareConfig{
					Rules: []HeaderRule{
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
				config: &headerMiddlewareConfig{
					Rules: []HeaderRule{
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			if err := m.Start(); (err != nil) != tt.wantErr {
				t.Errorf("headerMiddleware.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderMiddlewareStop(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			m.Stop()
		})
	}
}

func TestHeaderMiddlewareHandler(t *testing.T) {
	type fields struct {
		config  *headerMiddlewareConfig
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
			m := &headerMiddleware{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
			}
			got := m.Handler(tt.args.next)
			if tt.wantNil && got != nil {
				t.Errorf("headerMiddleware.Handler() = %v, want %v", got, nil)
			}
		})
	}
}
