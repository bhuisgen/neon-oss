// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"
)

type testIndexRendererFileInfo struct {
	name     string
	size     int64
	fileMode os.FileMode
	modTime  time.Time
	isDir    bool
	sys      any
}

func (fi testIndexRendererFileInfo) Name() string {
	return fi.name
}

func (fi testIndexRendererFileInfo) Size() int64 {
	return fi.size
}

func (fi testIndexRendererFileInfo) Mode() os.FileMode {
	return fi.fileMode
}
func (fi testIndexRendererFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi testIndexRendererFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi testIndexRendererFileInfo) Sys() any {
	return fi.sys
}

type testIndexRendererFetcher struct {
	errFetch                           bool
	errFetchOnlyForNames               []string
	exists                             bool
	get                                map[string][]byte
	errGet                             bool
	createResourceFromTemplateResource *Resource
	errCreateResourceFromTemplate      bool
}

func (t testIndexRendererFetcher) Fetch(ctx context.Context, name string) error {
	if t.errFetch {
		return errors.New("test error")
	}
	for _, n := range t.errFetchOnlyForNames {
		if n == name {
			return errors.New("test error")
		}
	}
	return nil
}

func (t testIndexRendererFetcher) Exists(name string) bool {
	return t.exists
}

func (t testIndexRendererFetcher) Get(name string) ([]byte, error) {
	if t.errGet {
		return nil, errors.New("test error")
	}
	return t.get[name], nil
}

func (t testIndexRendererFetcher) Register(r Resource) {
}

func (t testIndexRendererFetcher) Unregister(name string) {
}

func (t testIndexRendererFetcher) CreateResourceFromTemplate(template string, resource string, params map[string]string,
	headers map[string]string) (*Resource, error) {
	if t.errCreateResourceFromTemplate {
		return nil, errors.New("test error")
	}
	return t.createResourceFromTemplateResource, nil
}

type testIndexRendererResponseWriter struct {
	header http.Header
}

func (w testIndexRendererResponseWriter) Header() http.Header {
	return w.header
}

func (w testIndexRendererResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testIndexRendererResponseWriter) WriteHeader(statusCode int) {
}

func TestCreateIndexRenderer(t *testing.T) {
	type args struct {
		config  *IndexRendererConfig
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config:  &IndexRendererConfig{},
				fetcher: &testIndexRendererFetcher{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateIndexRenderer(tt.args.config, tt.args.fetcher)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIndexRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestIndexRendererInitialize(t *testing.T) {
	type fields struct {
		config      *IndexRendererConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		html        *[]byte
		htmlInfo    *time.Time
		bundle      *string
		bundleInfo  *time.Time
		bufferPool  BufferPool
		vmPool      VMPool
		cache       Cache
		fetcher     Fetcher
		next        Renderer
		osStat      func(name string) (fs.FileInfo, error)
		osReadFile  func(name string) ([]byte, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:   "dist/index.html",
					Bundle: stringPtr("dist/bundle.js"),
					Rules: []IndexRule{
						{
							Path: "^/",
						},
					},
				},
				logger:  log.Default(),
				regexps: []*regexp.Regexp{},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
		},
		{
			name: "rule regexp compile error",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:   "dist/index.html",
					Bundle: stringPtr("dist/bundle.js"),
					Rules: []IndexRule{
						{
							Path: "(",
						},
					},
				},
				logger:  log.Default(),
				regexps: []*regexp.Regexp{},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte{}, errors.New("test error")
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &indexRenderer{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				html:        tt.fields.html,
				htmlInfo:    tt.fields.htmlInfo,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				bufferPool:  tt.fields.bufferPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				fetcher:     tt.fields.fetcher,
				next:        tt.fields.next,
				osStat:      tt.fields.osStat,
				osReadFile:  tt.fields.osReadFile,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			if err := r.initialize(); (err != nil) != tt.wantErr {
				t.Errorf("indexRenderer.initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIndexRendererHandle(t *testing.T) {
	ctx := context.WithValue(context.Background(), ServerHandlerContextKeyRequestID{}, "test")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Error("failed to create request")
	}

	type fields struct {
		config      *IndexRendererConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		html        *[]byte
		htmlInfo    *time.Time
		bundle      *string
		bundleInfo  *time.Time
		bufferPool  BufferPool
		vmPool      VMPool
		cache       Cache
		fetcher     Fetcher
		next        Renderer
		osStat      func(name string) (fs.FileInfo, error)
		osReadFile  func(name string) ([]byte, error)
		jsonMarshal func(v any) ([]byte, error)
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
			name: "render",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:    "dist/index.html",
					Bundle:  stringPtr("dist/bundle.js"),
					Timeout: 1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				w:    testErrorRendererResponseWriter{},
				req:  req,
				info: &ServerInfo{},
			},
		},
		{
			name: "error render",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:    "dist/index.html",
					Bundle:  stringPtr("dist/bundle.js"),
					Timeout: 1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => {	while(true){} })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				w:    testErrorRendererResponseWriter{},
				req:  req,
				info: &ServerInfo{},
			},
		},
		{
			name: "redirect",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:    "dist/index.html",
					Bundle:  stringPtr("dist/bundle.js"),
					Timeout: 1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.redirect("http://external", 302); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				w:    testErrorRendererResponseWriter{http.Header{}},
				req:  req,
				info: &ServerInfo{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &indexRenderer{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				html:        tt.fields.html,
				htmlInfo:    tt.fields.htmlInfo,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				bufferPool:  tt.fields.bufferPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				fetcher:     tt.fields.fetcher,
				next:        tt.fields.next,
				osStat:      tt.fields.osStat,
				osReadFile:  tt.fields.osReadFile,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestIndexRendererNext(t *testing.T) {
	type fields struct {
		config      *IndexRendererConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		html        *[]byte
		htmlInfo    *time.Time
		bundle      *string
		bundleInfo  *time.Time
		bufferPool  BufferPool
		vmPool      VMPool
		cache       Cache
		fetcher     Fetcher
		next        Renderer
		osStat      func(name string) (fs.FileInfo, error)
		osReadFile  func(name string) ([]byte, error)
		jsonMarshal func(v any) ([]byte, error)
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
			r := &indexRenderer{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				html:        tt.fields.html,
				htmlInfo:    tt.fields.htmlInfo,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				bufferPool:  tt.fields.bufferPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				fetcher:     tt.fields.fetcher,
				next:        tt.fields.next,
				osStat:      tt.fields.osStat,
				osReadFile:  tt.fields.osReadFile,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			r.Next(tt.args.renderer)
		})
	}
}

func TestIndexRendererRender(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	ctx := context.WithValue(context.Background(), ServerHandlerContextKeyRequestID{}, "test")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Error("failed to create request")
	}

	re1, err := regexp.Compile("^/test1/(?P<slug>.+)/?")
	if err != nil {
		t.Error("failed to compile regular expression")
	}
	re2, err := regexp.Compile("^/test2/(.+)/?")
	if err != nil {
		t.Error("failed to compile regular expression")
	}

	req1, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test1/value", nil)
	if err != nil {
		t.Error("failed to create request")
	}

	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test2/value", nil)
	if err != nil {
		t.Error("failed to create request")
	}

	type fields struct {
		config      *IndexRendererConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		html        *[]byte
		htmlInfo    *time.Time
		bundle      *string
		bundleInfo  *time.Time
		bufferPool  BufferPool
		vmPool      VMPool
		cache       Cache
		fetcher     Fetcher
		next        Renderer
		osStat      func(name string) (fs.FileInfo, error)
		osReadFile  func(name string) ([]byte, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	type args struct {
		req  *http.Request
		info *ServerInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *indexRender
		wantW   string
		wantErr bool
	}{
		{
			name: "html",
			fields: fields{
				config: &IndexRendererConfig{
					HTML: "dist/index.html",
				},
				logger:  log.Default(),
				regexps: []*regexp.Regexp{},
				cache:   newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte("html"), nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "html",
		},
		{
			name: "html error stat file",
			fields: fields{
				config: &IndexRendererConfig{
					HTML: "dist/index.html",
				},
				logger: log.Default(),
				osStat: func(name string) (fs.FileInfo, error) {
					if name == "dist/index.html" {
						return nil, errors.New("test error")
					}
					return testConfigFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte{}, errors.New("test error")
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			wantErr: true,
		},
		{
			name: "html error read file",
			fields: fields{
				config: &IndexRendererConfig{
					HTML: "dist/index.html",
				},
				logger: log.Default(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte{}, errors.New("test error")
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			wantErr: true,
		},
		{
			name: "bundle",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div></body>",
		},
		{
			name: "error bundle stat file",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					if name == "dist/bundle.js" {
						return nil, errors.New("test error")
					}
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte{}, errors.New("test error")
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			wantErr: true,
		},
		{
			name: "error bundle read file",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte{}, errors.New("test error")
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			wantErr: true,
		},
		{
			name: "error vm configure",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: nil,
			},
			wantErr: true,
		},
		{
			name: "error vm execute",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => {	while(true){} })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			wantErr: true,
		},
		{
			name: "bundle render",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div></body>",
		},
		{
			name: "bundle redirect",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.redirect("http://external", 302); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Redirect:       true,
				RedirectURL:    "http://external",
				RedirectStatus: 302,
			},
			wantW: "",
		},
		{
			name: "state with a named capturing group regex",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					State:     "state",
					Timeout:   1,
					Rules: []IndexRule{
						{
							Path: "/test1/(?P<slug>.+)/?",
							State: []IndexRuleStateEntry{
								{
									Key:      "test1-$slug",
									Resource: "resource-test1-$slug",
									Export:   boolPtr(true),
								},
							},
							Last: true,
						},
					},
				},
				logger:     log.Default(),
				regexps:    []*regexp.Regexp{re1},
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher: testIndexRendererFetcher{
					get: map[string][]byte{
						"resource-test1-value": []byte(`{"data": {"id": 1, "string": "test", "float": -1.00, "bool": true}}`),
					},
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return json.Marshal(v)
				},
			},
			args: args{
				req:  req1,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div>" +
				"<script id=\"state\" type=\"application/json\">" +
				"{\"test1-value\":{\"loading\":false,\"error\":\"\",\"response\":" +
				"\"{\\\"data\\\": {\\\"id\\\": 1, \\\"string\\\": \\\"test\\\", \\\"float\\\": -1.00, \\\"bool\\\": true}}\"" +
				"}}" +
				"</script></body>",
		},
		{
			name: "state with an indexed capturing group regex",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					State:     "state",
					Timeout:   1,
					Rules: []IndexRule{
						{
							Path: "/test2/(.+)/?",
							State: []IndexRuleStateEntry{
								{
									Key:      "test2-$1",
									Resource: "resource-test2-$1",
									Export:   boolPtr(true),
								},
							},
							Last: true,
						},
					},
				},
				logger:     log.Default(),
				regexps:    []*regexp.Regexp{re2},
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher: testIndexRendererFetcher{
					get: map[string][]byte{
						"resource-test2-value": []byte(`{"data": {"id": 1, "string": "test", "float": -1.00, "bool": true}}`),
					},
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return json.Marshal(v)
				},
			},
			args: args{
				req:  req2,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div>" +
				"<script id=\"state\" type=\"application/json\">" +
				"{\"test2-value\":{\"loading\":false,\"error\":\"\",\"response\":" +
				"\"{\\\"data\\\": {\\\"id\\\": 1, \\\"string\\\": \\\"test\\\", \\\"float\\\": -1.00, \\\"bool\\\": true}}\"" +
				"}}" +
				"</script></body>",
		},
		{
			name: "state without match",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					State:     "state",
					Timeout:   1,
					Rules: []IndexRule{
						{
							Path: "/test1/(.+)/?",
							State: []IndexRuleStateEntry{
								{
									Key:      "test2-$1",
									Resource: "resource-test2-$1",
									Export:   boolPtr(true),
								},
							},
							Last: true,
						},
					},
				},
				logger:     log.Default(),
				regexps:    []*regexp.Regexp{re1},
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher: testIndexRendererFetcher{
					get: map[string][]byte{
						"resource-test2-value": []byte(`{"data": {"id": 1, "string": "test", "float": -1.00, "bool": true}}`),
					},
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div></body>",
		},
		{
			name: "state error fetcher get",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					State:     "state",
					Timeout:   1,
					Rules: []IndexRule{
						{
							Path: "/test1/(?P<slug>.+)/?",
							State: []IndexRuleStateEntry{
								{
									Key:      "test1-$slug",
									Resource: "resource-test1-$slug",
									Export:   boolPtr(true),
								},
							},
							Last: true,
						},
					},
				},
				logger:     log.Default(),
				regexps:    []*regexp.Regexp{re1},
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher: testIndexRendererFetcher{
					exists: true,
					errGet: true,
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return json.Marshal(v)
				},
			},
			args: args{
				req:  req1,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div>" +
				"<script id=\"state\" type=\"application/json\">" +
				"{\"test1-value\":{\"loading\":true,\"error\":\"\",\"response\":\"\"" +
				"}}" +
				"</script></body>",
		},
		{
			name: "state error fetcher exists",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					State:     "state",
					Timeout:   1,
					Rules: []IndexRule{
						{
							Path: "/test1/(?P<slug>.+)/?",
							State: []IndexRuleStateEntry{
								{
									Key:      "test1-$slug",
									Resource: "resource-test1-$slug",
									Export:   boolPtr(true),
								},
							},
							Last: true,
						},
					},
				},
				logger:     log.Default(),
				regexps:    []*regexp.Regexp{re1, re2},
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher: testIndexRendererFetcher{
					errGet: true,
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return json.Marshal(v)
				},
			},
			args: args{
				req:  req1,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div>" +
				"<script id=\"state\" type=\"application/json\">" +
				"{\"test1-value\":{\"loading\":false,\"error\":\"unknown resource\",\"response\":\"\"" +
				"}}" +
				"</script></body>",
		},
		{
			name: "state error json unmarshal",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					State:     "state",
					Timeout:   1,
					Rules: []IndexRule{
						{
							Path: "/test1/(?P<slug>.+)/?",
							State: []IndexRuleStateEntry{
								{
									Key:      "test1-$slug",
									Resource: "resource-test1-$slug",
									Export:   boolPtr(true),
								},
							},
							Last: true,
						},
					},
				},
				logger:     log.Default(),
				regexps:    []*regexp.Regexp{re1, re2},
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher: testIndexRendererFetcher{
					get: map[string][]byte{
						"resource-test2-value": []byte(`{"data": {"id": 1, "string": "test", "float": -1.00, "bool": true}}`),
					},
				},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, errors.New("test error")
				},
			},
			args: args{
				req:  req1,
				info: &ServerInfo{},
			},
			wantErr: true,
		},
		{
			name: "bundle with custom container",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "test",
					State:     "state",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="test"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"test\"><p>test</p></div></body>",
		},
		{
			name: "bundle with title",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200);` +
							` serverResponse.setTitle("test"); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8><title>test</title></head><body><div id=\"root\"><p>test</p></div></body>",
		},
		{
			name: "bundle with meta",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200);` +
							` serverResponse.setMeta("test", new Map([["name", "test"]])); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8><meta id=\"test\" name=\"test\"></head><body><div id=\"root\"><p>test</p></div></body>",
		},
		{
			name: "bundle with link",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200);` +
							` serverResponse.setLink("test", new Map([["href", "test"],["rel", "test"]])); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8><link id=\"test\" href=\"test\" rel=\"test\"></head><body><div id=\"root\"><p>test</p></div></body>",
		},
		{
			name: "bundle with script",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					Timeout:   1,
				},
				logger:     log.Default(),
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				fetcher:    testIndexRendererFetcher{},
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200);` +
							` serverResponse.setScript("test", new Map([["type", "test"], ["children", "content"]])); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return []byte{}, nil
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8><script id=\"test\" type=\"test\">content</script></head><body><div id=\"root\"><p>test</p></div></body>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &indexRenderer{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				html:        tt.fields.html,
				htmlInfo:    tt.fields.htmlInfo,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				bufferPool:  tt.fields.bufferPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				fetcher:     tt.fields.fetcher,
				next:        tt.fields.next,
				osStat:      tt.fields.osStat,
				osReadFile:  tt.fields.osReadFile,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			w := &bytes.Buffer{}
			got, err := r.render(tt.args.req, tt.args.info, w)
			if (err != nil) != tt.wantErr {
				t.Errorf("indexRenderer.render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("indexRenderer.render() = %v, want %v", got, tt.want)
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("indexRenderer.render() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestIndexRendererRender_Debug(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	DEBUG = true
	defer func() {
		DEBUG = false
	}()
	ctx := context.WithValue(context.Background(), ServerHandlerContextKeyRequestID{}, "test")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Error("failed to create request")
	}

	re, err := regexp.Compile("^/(?P<slug>.+)/?")
	if err != nil {
		t.Error("failed to compile regular expression")
	}

	type fields struct {
		config      *IndexRendererConfig
		logger      *log.Logger
		regexps     []*regexp.Regexp
		html        *[]byte
		htmlInfo    *time.Time
		bundle      *string
		bundleInfo  *time.Time
		bufferPool  BufferPool
		vmPool      VMPool
		cache       Cache
		fetcher     Fetcher
		next        Renderer
		osStat      func(name string) (fs.FileInfo, error)
		osReadFile  func(name string) ([]byte, error)
		jsonMarshal func(v any) ([]byte, error)
	}
	type args struct {
		req  *http.Request
		info *ServerInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *indexRender
		wantW   string
		wantErr bool
	}{
		{
			name: "state with debug",
			fields: fields{
				config: &IndexRendererConfig{
					HTML:      "dist/index.html",
					Bundle:    stringPtr("dist/bundle.js"),
					Container: "root",
					State:     "state",
					Timeout:   1,
					Rules: []IndexRule{
						{
							Path: "/(?P<slug>.+)/?",
							State: []IndexRuleStateEntry{
								{
									Key:      "test-$slug",
									Resource: "resource-$slug",
									Export:   boolPtr(true),
								},
							},
							Last: true,
						},
					},
				},
				logger:     log.Default(),
				regexps:    []*regexp.Regexp{re},
				bufferPool: newBufferPool(),
				vmPool:     newVMPool(1),
				cache:      newCache(),
				osStat: func(name string) (fs.FileInfo, error) {
					return testIndexRendererFileInfo{}, nil
				},
				fetcher: testIndexRendererFetcher{
					get: map[string][]byte{
						"resource-test": []byte(`{"data": {"id": 1, "string": "test", "float": -1.00, "bool": true}}`),
					},
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "dist/index.html" {
						return []byte(`<!doctype html><head><meta charset=utf-8></head><body><div id="root"></div></body>`), nil
					}
					if name == "dist/bundle.js" {
						return []byte(`(() => { serverResponse.render("<p>test</p>", 200); })();`), nil
					}
					return []byte{}, nil
				},
				jsonMarshal: func(v any) ([]byte, error) {
					return json.Marshal(v)
				},
			},
			args: args{
				req:  req,
				info: &ServerInfo{},
			},
			want: &indexRender{
				Status: 200,
			},
			wantW: "<!doctype html><head><meta charset=utf-8></head><body><div id=\"root\"><p>test</p></div>" +
				"<script id=\"state\" type=\"application/json\">" +
				"{\"test-test\":{\"loading\":false,\"error\":\"\",\"response\":" +
				"\"{\\\"data\\\": {\\\"id\\\": 1, \\\"string\\\": \\\"test\\\", \\\"float\\\": -1.00, \\\"bool\\\": true}}\"" +
				"}}" +
				"</script></body>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &indexRenderer{
				config:      tt.fields.config,
				logger:      tt.fields.logger,
				regexps:     tt.fields.regexps,
				html:        tt.fields.html,
				htmlInfo:    tt.fields.htmlInfo,
				bundle:      tt.fields.bundle,
				bundleInfo:  tt.fields.bundleInfo,
				bufferPool:  tt.fields.bufferPool,
				vmPool:      tt.fields.vmPool,
				cache:       tt.fields.cache,
				fetcher:     tt.fields.fetcher,
				next:        tt.fields.next,
				osStat:      tt.fields.osStat,
				osReadFile:  tt.fields.osReadFile,
				jsonMarshal: tt.fields.jsonMarshal,
			}
			w := &bytes.Buffer{}
			got, err := r.render(tt.args.req, tt.args.info, w)
			if (err != nil) != tt.wantErr {
				t.Errorf("indexRenderer.render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("indexRenderer.render() = %v, want %v", got, tt.want)
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("indexRenderer.render() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
