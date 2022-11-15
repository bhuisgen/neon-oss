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

type testRewriteRendererNextRenderer struct{}

func (r testRewriteRendererNextRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
}

func (r testRewriteRendererNextRenderer) Next(renderer Renderer) {
}

type testRewriteRendererResponseWriter struct {
	header http.Header
}

func (w testRewriteRendererResponseWriter) Header() http.Header {
	return w.header
}

func (w testRewriteRendererResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testRewriteRendererResponseWriter) WriteHeader(statusCode int) {
}

func TestCreateRewriteRenderer(t *testing.T) {
	type args struct {
		config *RewriteRendererConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &RewriteRendererConfig{},
			},
		},
		{
			name: "invalid regexp",
			args: args{
				config: &RewriteRendererConfig{
					Rules: []RewriteRule{
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
			_, err := CreateRewriteRenderer(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRewriteRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRewriteRendererInitialize(t *testing.T) {
	type fields struct {
		config  *RewriteRendererConfig
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
				config: &RewriteRendererConfig{
					Rules: []RewriteRule{
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
				config: &RewriteRendererConfig{
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
			r := &rewriteRenderer{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
				next:    tt.fields.next,
			}
			if err := r.initialize(); (err != nil) != tt.wantErr {
				t.Errorf("rewriteRenderer.initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRewriteRendererHandle(t *testing.T) {
	re, err := regexp.Compile("/")
	if err != nil {
		t.Error("failed to compile regular expression")
	}

	type fields struct {
		config  *RewriteRendererConfig
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
				config: &RewriteRendererConfig{},
				logger: log.Default(),
				next:   testRewriteRendererNextRenderer{},
			},
			args: args{
				w: testRewriteRendererResponseWriter{
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
			name: "rewrite",
			fields: fields{
				config: &RewriteRendererConfig{
					Rules: []RewriteRule{
						{
							Path:        "/",
							Replacement: "/rewrite1",
						},
						{
							Path:        "/",
							Replacement: "/rewrite2",
							Last:        true,
						},
					},
				},
				logger:  log.Default(),
				regexps: []*regexp.Regexp{re, re},
				next:    testRewriteRendererNextRenderer{},
			},
			args: args{
				w: testRewriteRendererResponseWriter{
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
			name: "redirect",
			fields: fields{
				config: &RewriteRendererConfig{
					Rules: []RewriteRule{
						{
							Path:        "/",
							Replacement: "http://redirect",
							Flag:        stringPtr("redirect"),
						},
						{
							Path:        "/",
							Replacement: "http://redirect",
							Flag:        stringPtr("permanent"),
							Last:        true,
						},
					},
				},
				logger:  log.Default(),
				regexps: []*regexp.Regexp{re, re},
				next:    testRewriteRendererNextRenderer{},
			},
			args: args{
				w: testRewriteRendererResponseWriter{
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
			r := &rewriteRenderer{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
				next:    tt.fields.next,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestRewriteRendererNext(t *testing.T) {
	type fields struct {
		config  *RewriteRendererConfig
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
			r := &rewriteRenderer{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				regexps: tt.fields.regexps,
				next:    tt.fields.next,
			}
			r.Next(tt.args.renderer)
		})
	}
}
