// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"log"
	"net/http"
	"net/url"
	"testing"
)

type testErrorRendererResponseWriter struct {
	header http.Header
}

func (w testErrorRendererResponseWriter) Header() http.Header {
	return w.header
}

func (w testErrorRendererResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testErrorRendererResponseWriter) WriteHeader(statusCode int) {
}

func TestCreateErrorRenderer(t *testing.T) {
	type args struct {
		config *ErrorRendererConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &ErrorRendererConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateErrorRenderer(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateErrorRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestErrorRendererHandle(t *testing.T) {
	type fields struct {
		config *ErrorRendererConfig
		logger *log.Logger
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
			name: "default",
			fields: fields{
				config: &ErrorRendererConfig{},
				logger: log.Default(),
			},
			args: args{
				w: testErrorRendererResponseWriter{},
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
			r := &errorRenderer{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestErrorRendererNext(t *testing.T) {
	type fields struct {
		config *ErrorRendererConfig
		logger *log.Logger
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
			r := &errorRenderer{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			r.Next(tt.args.renderer)
		})
	}
}
