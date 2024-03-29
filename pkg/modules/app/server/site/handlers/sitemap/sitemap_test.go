package sitemap

import (
	_ "embed"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"text/template"

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

type testSitemapHandlerServerSite struct {
	err bool
}

func (s testSitemapHandlerServerSite) Name() string {
	return "test"
}

func (s testSitemapHandlerServerSite) Listeners() []string {
	return nil
}

func (s testSitemapHandlerServerSite) Hosts() []string {
	return nil
}

func (s testSitemapHandlerServerSite) IsDefault() bool {
	return false
}

func (s testSitemapHandlerServerSite) Store() core.Store {
	return nil
}

func (s testSitemapHandlerServerSite) Fetcher() core.Fetcher {
	return nil
}

func (s testSitemapHandlerServerSite) Loader() core.Loader {
	return nil
}

func (s testSitemapHandlerServerSite) Server() core.Server {
	return nil
}

func (s testSitemapHandlerServerSite) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

func (s testSitemapHandlerServerSite) RegisterHandler(handler http.Handler) error {
	if s.err {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerSite = (*testSitemapHandlerServerSite)(nil)

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
		logger               *slog.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                *sitemapHandlerCache
		muCache              *sync.RWMutex
		site                 core.ServerSite
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
				muCache:              tt.fields.muCache,
				site:                 tt.fields.site,
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

func TestSitemapHandlerInit(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *slog.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                *sitemapHandlerCache
		muCache              *sync.RWMutex
		site                 core.ServerSite
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
			name: "minimal sitemap index",
			fields: fields{
				logger: slog.Default(),
			},
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
			fields: fields{
				logger: slog.Default(),
			},
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
			fields: fields{
				logger: slog.Default(),
			},
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
			fields: fields{
				logger: slog.Default(),
			},
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
			fields: fields{
				logger: slog.Default(),
			},
			args: args{
				config: map[string]interface{}{
					"Root":     "",
					"CacheTTL": 0,
					"Kind":     "",
				},
			},
			wantErr: true,
		},
		{
			name: "missing sitemap index entry",
			fields: fields{
				logger: slog.Default(),
			},
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemapIndex",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid sitemap index entry type",
			fields: fields{
				logger: slog.Default(),
			},
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
			wantErr: true,
		},
		{
			name: "invalid sitemap index values",
			fields: fields{
				logger: slog.Default(),
			},
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
			wantErr: true,
		},
		{
			name: "missing sitemap entry",
			fields: fields{
				logger: slog.Default(),
			},
			args: args{
				config: map[string]interface{}{
					"Root": "http://localhost",
					"Kind": "sitemap",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid sitemap values",
			fields: fields{
				logger: slog.Default(),
			},
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
				muCache:              tt.fields.muCache,
				site:                 tt.fields.site,
			}
			if err := h.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSitemapHandlerRegister(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *slog.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                *sitemapHandlerCache
		muCache              *sync.RWMutex
		site                 core.ServerSite
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
				site: testSitemapHandlerServerSite{},
			},
		},
		{
			name: "error register",
			args: args{
				site: testSitemapHandlerServerSite{
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
				muCache:              tt.fields.muCache,
				site:                 tt.fields.site,
			}
			if err := h.Register(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSitemapHandlerStart(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *slog.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                *sitemapHandlerCache
		muCache              *sync.RWMutex
		site                 core.ServerSite
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
				muCache:              tt.fields.muCache,
				site:                 tt.fields.site,
			}
			if err := h.Start(); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSitemapHandlerStop(t *testing.T) {
	type fields struct {
		config               *sitemapHandlerConfig
		logger               *slog.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                *sitemapHandlerCache
		muCache              *sync.RWMutex
		site                 core.ServerSite
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				muCache: &sync.RWMutex{},
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
				muCache:              tt.fields.muCache,
				site:                 tt.fields.site,
			}
			if err := h.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("sitemapHandler.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
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
		logger               *slog.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		rwPool               render.RenderWriterPool
		cache                *sitemapHandlerCache
		muCache              *sync.RWMutex
		site                 core.ServerSite
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
				logger:               slog.Default(),
				templateSitemapIndex: tmplSitemapIndex,
				templateSitemap:      tmplSitemap,
				rwPool:               render.NewRenderWriterPool(),
				cache:                &sitemapHandlerCache{},
				muCache:              &sync.RWMutex{},
			},
			args: args{
				w: testSitemapHandlerResponseWriter{},
				r: &http.Request{
					Method: http.MethodGet,
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
				muCache:              tt.fields.muCache,
				site:                 tt.fields.site,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
