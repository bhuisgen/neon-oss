// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"testing"
)

type testDefaultRendererNextRenderer struct{}

func (r testDefaultRendererNextRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
}

func (r testDefaultRendererNextRenderer) Next(renderer Renderer) {
}

func TestCreateDefaultRenderer(t *testing.T) {
	type args struct {
		config *DefaultRendererConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &DefaultRendererConfig{
					File: "/data/default.html",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateDefaultRenderer(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDefaultRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDefaultRendererHandle(t *testing.T) {
	type fields struct {
		config     *DefaultRendererConfig
		logger     *log.Logger
		cache      Cache
		next       Renderer
		osReadFile func(name string) ([]byte, error)
	}
	type args struct {
		w    http.ResponseWriter
		req  *http.Request
		info *ServerInfo
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "required",
			fields: fields{
				config: &DefaultRendererConfig{
					File: "/data/index.html",
				},
				logger: log.Default(),
				cache:  newCache(),
				next:   testDefaultRendererNextRenderer{},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				w: testIndexRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "cache",
			fields: fields{
				config: &DefaultRendererConfig{
					File:     "/data/index.html",
					Cache:    true,
					CacheTTL: 60,
				},
				logger: log.Default(),
				cache:  newCache(),
				next:   testDefaultRendererNextRenderer{},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				w: testIndexRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "error render",
			fields: fields{
				config: &DefaultRendererConfig{
					File: "/data/index.html",
				},
				logger: log.Default(),
				cache:  newCache(),
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, errors.New("test error")
				},
			},
			args: args{
				w: testIndexRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/",
					},
				},
				info: &ServerInfo{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultRenderer{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				cache:      tt.fields.cache,
				next:       tt.fields.next,
				osReadFile: tt.fields.osReadFile,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestDefaultRendererNext(t *testing.T) {
	type fields struct {
		config     *DefaultRendererConfig
		logger     *log.Logger
		cache      Cache
		next       Renderer
		osReadFile func(name string) ([]byte, error)
	}
	type args struct {
		renderer Renderer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			args: args{
				renderer: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultRenderer{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				cache:      tt.fields.cache,
				next:       tt.fields.next,
				osReadFile: tt.fields.osReadFile,
			}
			r.Next(tt.args.renderer)
		})
	}
}
