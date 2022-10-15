// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"log"
	"net/http"
	"net/url"
	"regexp"
	"testing"
)

type testHeaderRendererNextRenderer struct{}

func (r testHeaderRendererNextRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
}

func (r testHeaderRendererNextRenderer) Next(renderer Renderer) {
}

type testHeaderRendererResponseWriter struct {
	header http.Header
}

func (w testHeaderRendererResponseWriter) Header() http.Header {
	return w.header
}

func (w testHeaderRendererResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testHeaderRendererResponseWriter) WriteHeader(statusCode int) {
}

func TestCreateHeaderRenderer(t *testing.T) {
	type args struct {
		config *HeaderRendererConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &HeaderRendererConfig{},
			},
		},
		{
			name: "invalid regexp",
			args: args{
				config: &HeaderRendererConfig{
					Rules: []HeaderRule{
						{
							Path: "/test",
						},
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
			_, err := CreateHeaderRenderer(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateHeaderRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestHeaderRendererInitialize(t *testing.T) {
	type fields struct {
		config  *HeaderRendererConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
		next    Renderer
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &HeaderRendererConfig{
					Rules: []HeaderRule{
						{
							Path: "/test",
						},
					},
				},
			},
		},
		{
			name: "invalid regexp",
			fields: fields{
				config: &HeaderRendererConfig{
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
			r := &headerRenderer{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
				next:    tt.fields.next,
			}
			if err := r.initialize(); (err != nil) != tt.wantErr {
				t.Errorf("headerRenderer.initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderRendererHandle(t *testing.T) {
	re, err := regexp.Compile("/")
	if err != nil {
		t.Error("failed to compile regular expression")
	}

	type fields struct {
		config  *HeaderRendererConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
		next    Renderer
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
				config: &HeaderRendererConfig{},
				logger: log.Default(),
				next:   testHeaderRendererNextRenderer{},
			},
			args: args{
				w: testHeaderRendererResponseWriter{
					header: http.Header{},
				},
				req: &http.Request{
					URL: &url.URL{
						Path: "/",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "rules",
			fields: fields{
				config: &HeaderRendererConfig{
					Rules: []HeaderRule{
						{
							Path: "/",
							Set: map[string]string{
								"header1": "value1",
							},
						},
					},
				},
				logger:  log.Default(),
				regexps: []*regexp.Regexp{re},
				next:    testHeaderRendererNextRenderer{},
			},
			args: args{
				w: testHeaderRendererResponseWriter{
					header: http.Header{},
				},
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
			r := &headerRenderer{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
				next:    tt.fields.next,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestHeaderRendererNext(t *testing.T) {
	type fields struct {
		config  *HeaderRendererConfig
		logger  *log.Logger
		regexps []*regexp.Regexp
		next    Renderer
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &headerRenderer{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
				next:    tt.fields.next,
			}
			r.Next(tt.args.renderer)
		})
	}
}
