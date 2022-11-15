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

type testStaticRendererNextRenderer struct{}

func (r testStaticRendererNextRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
}

func (r testStaticRendererNextRenderer) Next(renderer Renderer) {
}

type testStaticRendererResponseWriter struct {
	header http.Header
}

func (w testStaticRendererResponseWriter) Header() http.Header {
	return w.header
}

func (w testStaticRendererResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testStaticRendererResponseWriter) WriteHeader(statusCode int) {
}

type testStaticFileSystem struct {
	exists    bool
	openFile  http.File
	openError bool
}

func (fs testStaticFileSystem) Exists(name string) bool {
	return fs.exists
}

func (fs *testStaticFileSystem) Open(name string) (http.File, error) {
	if fs.openError {
		return nil, errors.New("test error")
	}
	return fs.openFile, nil
}

type testStaticHandler struct{}

func (h testStaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

func TestCreateStaticRenderer(t *testing.T) {
	type args struct {
		config *StaticRendererConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &StaticRendererConfig{
					Dir: "/dist/static",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateStaticRenderer(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateStaticRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestStaticRendererHandle(t *testing.T) {
	type fields struct {
		config        *StaticRendererConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		next          Renderer
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
				config: &StaticRendererConfig{
					Dir: "dist/static",
				},
				logger:        log.Default(),
				staticFS:      &testStaticFileSystem{},
				staticHandler: testStaticHandler{},
				next:          testStaticRendererNextRenderer{},
			},
			args: args{
				w: testStaticRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/index.html",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "static file",
			fields: fields{
				config: &StaticRendererConfig{
					Dir: "dist/static",
				},
				logger: log.Default(),
				staticFS: &testStaticFileSystem{
					exists: true,
				},
				staticHandler: testStaticHandler{},
				next:          testStaticRendererNextRenderer{},
			},
			args: args{
				w: testStaticRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/image.jpg",
					},
				},
				info: &ServerInfo{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &staticRenderer{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				next:          tt.fields.next,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestStaticRendererNext(t *testing.T) {
	type fields struct {
		config        *StaticRendererConfig
		logger        *log.Logger
		staticFS      StaticFileSystem
		staticHandler http.Handler
		next          Renderer
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
			r := &staticRenderer{
				config:        tt.fields.config,
				logger:        tt.fields.logger,
				staticFS:      tt.fields.staticFS,
				staticHandler: tt.fields.staticHandler,
				next:          tt.fields.next,
			}
			r.Next(tt.args.renderer)
		})
	}
}
