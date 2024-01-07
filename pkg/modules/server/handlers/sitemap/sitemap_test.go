// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	_ "embed"
	"errors"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"text/template"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/cache/memory"
	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

type testSitemapHandlerServer struct {
	err bool
}

func (s testSitemapHandlerServer) Name() string {
	return "test"
}

func (s testSitemapHandlerServer) Listeners() []string {
	return nil
}

func (s testSitemapHandlerServer) Hosts() []string {
	return nil
}

func (s testSitemapHandlerServer) Store() core.Store {
	return nil
}

func (s testSitemapHandlerServer) Fetcher() core.Fetcher {
	return nil
}

func (s testSitemapHandlerServer) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testSitemapHandlerServer) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.Server = (*testSitemapHandlerServer)(nil)

type testSitemapHandlerResponseWriter struct {
	header http.Header
}

func (w testSitemapHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testSitemapHandlerResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testSitemapHandlerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testSitemapHandlerResponseWriter)(nil)

func TestSitemapHandlerModuleInfo(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          sitemapModuleID,
				NewInstance: func() module.Module { return &sitemapHandler{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			got := h.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("sitemapHandler.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("sitemapHandler.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestSitemapHandlerCheck(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
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
			name: "minimal sitemap index",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemapIndex",
					"SitemapIndex": []map[string]interface{}{
						{
							"Name": "test",
							"Type": "static",
							"Static": map[string]interface{}{
								"Loc": "http://localhost/sitemap_test.xml",
							},
						},
					},
				},
			},
		},
		{
			name: "full sitemap index",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemapIndex",
					"SitemapIndex": []map[string]interface{}{
						{
							"Name": "test",
							"Type": "static",
							"Static": map[string]interface{}{
								"Loc":        "http://localhost/sitemap_test.xml",
								"Changefreq": "always",
								"Priority":   0.5,
							},
						},
					},
				},
			},
		},
		{
			name: "minimal sitemap",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemap",
					"Sitemap": []map[string]interface{}{
						{
							"Name": "home",
							"Type": "static",
							"Static": map[string]interface{}{
								"Loc": "/",
							},
						},
					},
				},
			},
		},
		{
			name: "full sitemap",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemap",
					"Sitemap": []map[string]interface{}{
						{
							"Name": "home",
							"Type": "static",
							"Static": map[string]interface{}{
								"Loc":        "/",
								"Changefreq": "always",
								"Priority":   0.5,
							},
						},
						{
							"Name": "posts",
							"Type": "list",
							"List": map[string]interface{}{
								"Resource":    "resource",
								"Filter":      "$.results",
								"ItemLoc":     "$.loc",
								"ItemLastmod": "$.lastmod",
								"ItemIgnore":  "$.ignore",
								"Changefreq":  "always",
								"Priority":    0.5,
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
					"Root":     "",
					"CacheTTL": -1,
					"Kind":     "",
				},
			},
			want: []string{
				"option 'Root', missing option or value",
				"option 'CacheTTL', invalid value '-1'",
				"option 'Kind', missing option or value",
			},
			wantErr: true,
		},
		{
			name: "missing sitemap index entry",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemapIndex",
				},
			},
			want: []string{
				"sitemapIndex entry is missing",
			},
			wantErr: true,
		},
		{
			name: "invalid sitemap index entry type",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemapIndex",
					"SitemapIndex": []map[string]interface{}{
						{
							"Name": "test",
							"Type": "",
						},
					},
				},
			},
			want: []string{
				"sitemapIndex entry option 'Type', missing option or value",
			},
			wantErr: true,
		},
		{
			name: "invalid sitemap index values",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemapIndex",
					"SitemapIndex": []map[string]interface{}{
						{
							"Name": "test",
							"Type": "static",
							"Static": map[string]interface{}{
								"Loc": "",
							},
						},
					},
				},
			},
			want: []string{
				"sitemapIndex static entry option 'Loc', missing option or value",
			},
			wantErr: true,
		},
		{
			name: "missing sitemap entry",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemap",
				},
			},
			want: []string{
				"sitemap entry is missing",
			},
			wantErr: true,
		},
		{
			name: "invalid sitemap values",
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemap",
					"Sitemap": []map[string]interface{}{
						{
							"Name": "",
							"Type": "static",
							"Static": map[string]interface{}{
								"Loc":        "",
								"Lastmod":    "",
								"Changefreq": "",
								"Priority":   -1,
							},
						},
						{
							"Name": "",
							"Type": "list",
							"List": map[string]interface{}{
								"Resource":    "",
								"Filter":      "",
								"ItemLoc":     "",
								"ItemLastmod": "",
								"ItemIgnore":  "",
								"Changefreq":  "",
								"Priority":    -1,
							},
						},
					},
				},
			},
			want: []string{
				"sitemap entry option 'Name', missing option or value",
				"sitemap static entry option 'Loc', missing option or value",
				"sitemap static entry option 'Lastmod', invalid value ''",
				"sitemap static entry option 'Changefreq', invalid value ''",
				"sitemap static entry option 'Priority', invalid value '-1.0'",
				"sitemap entry option 'Name', missing option or value",
				"sitemap list entry option 'Resource', missing option or value",
				"sitemap list entry option 'Filter', missing option or value",
				"sitemap list entry option 'ItemLoc', missing option or value",
				"sitemap list entry option 'ItemLastmod', invalid value ''",
				"sitemap list entry option 'ItemIgnore', invalid value ''",
				"sitemap list entry option 'Changefreq', invalid value ''",
				"sitemap list entry option 'Priority', invalid value '-1.0'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			got, err := h.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sitemapHandler.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSitemapHandlerLoad(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
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
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			if err := h.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSitemapHandlerRegister(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
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
				server: testSitemapHandlerServer{},
			},
		},
		{
			name: "error register",
			args: args{
				server: testSitemapHandlerServer{
					err: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			if err := h.Register(tt.args.server); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSitemapHandlerStart(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
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
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			if err := h.Start(); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSitemapHandlerMount(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
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
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			if err := h.Mount(); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSitemapHandlerUnmount(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
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
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			h.Unmount()
		})
	}
}

func TestSitemapHandlerStop(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				cache: memory.NewMemoryCache(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			h.Stop()
		})
	}
}

func TestSitemapHandlerServeHTTP(t *testing.T) {
	tmplSitemapIndex, err := template.New("sitemapIndex").Parse(sitemapTemplateSitemapIndex)
	if err != nil {
		t.Error(err)
	}

	tmplSitemap, err := template.New("sitemapIndex").Parse(sitemapTemplateSitemap)
	if err != nil {
		t.Error(err)
	}

	type fields struct {
		config               *sitemapHandlerConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                cache.Cache
		server               core.Server
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				config: &sitemapHandlerConfig{
					Cache:    boolPtr(true),
					CacheTTL: intPtr(60),
				},
				logger:               log.Default(),
				templateSitemapIndex: tmplSitemapIndex,
				templateSitemap:      tmplSitemap,
				rwPool:               render.NewRenderWriterPool(),
				cache:                memory.NewMemoryCache(),
			},
			args: args{
				w: testSitemapHandlerResponseWriter{},
				r: &http.Request{
					URL: &url.URL{
						Path: "/test",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &sitemapHandler{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				rwPool:               tt.fields.rwPool,
				cache:                tt.fields.cache,
				server:               tt.fields.server,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
