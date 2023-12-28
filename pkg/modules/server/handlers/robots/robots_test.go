// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Unauthorized copying of this file, via any medium is strictly prohibited.

package robots

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

type testRobotsHandlerServerRegistry struct {
	error bool
}

func (r testRobotsHandlerServerRegistry) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	if r.error {
		return errors.New("test error")
	}
	return nil
}

func (r testRobotsHandlerServerRegistry) RegisterHandler(handler http.Handler) error {
	if r.error {
		return errors.New("test error")
	}
	return nil
}

var _ core.ServerRegistry = (*testRobotsHandlerServerRegistry)(nil)

type testRobotsHandlerResponseWriter struct {
	header http.Header
}

func (w testRobotsHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testRobotsHandlerResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testRobotsHandlerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testRobotsHandlerResponseWriter)(nil)

func TestRobotsHandlerModuleInfo(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          robotsModuleID,
				NewInstance: func() module.Module { return &robotsHandler{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			got := h.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("robotsHandler.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("robotsHandler.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestRobotsHandlerCheck(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
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
					"Hosts":    []string{"test"},
					"Cache":    true,
					"CacheTTL": 60,
					"Sitemaps": []string{"http://test/sitemap.xml"},
				},
			},
		},
		{
			name: "invalid values",
			args: args{
				config: map[string]interface{}{
					"Hosts":    []string{""},
					"CacheTTL": -1,
					"Sitemaps": []string{""},
				},
			},
			want: []string{
				"option 'Hosts', missing option or value",
				"option 'CacheTTL', invalid value '-1'",
				"option 'Sitemaps', invalid value ''",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			got, err := h.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("robotsHandler.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("robotsHandler.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRobotsHandlerLoad(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
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
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			if err := h.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("robotsHandler.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRobotsHandlerRegister(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
	}
	type args struct {
		registry core.ServerRegistry
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
				registry: testRobotsHandlerServerRegistry{},
			},
		},
		{
			name: "error register",
			args: args{
				registry: testRobotsHandlerServerRegistry{
					error: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			if err := h.Register(tt.args.registry); (err != nil) != tt.wantErr {
				t.Errorf("robotsHandler.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRobotsHandlerStart(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
	}
	type args struct {
		store   core.Store
		fetcher core.Fetcher
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
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			if err := h.Start(tt.args.store, tt.args.fetcher); (err != nil) != tt.wantErr {
				t.Errorf("robotsHandler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRobotsHandlerMount(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
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
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			if err := h.Mount(); (err != nil) != tt.wantErr {
				t.Errorf("robotsHandler.Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRobotsHandlerUnmount(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
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
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			h.Unmount()
		})
	}
}

func TestRobotsHandlerStop(t *testing.T) {
	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				cache: cache.NewCache(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			h.Stop()
		})
	}
}

func TestRobotsHandlerServeHTTP(t *testing.T) {
	tmpl, err := template.New("robots").Parse(robotsTemplate)
	if err != nil {
		t.Error(err)
	}

	type fields struct {
		config   *robotsHandlerConfig
		logger   *log.Logger
		template *template.Template
		rwPool   render.RenderWriterPool
		cache    cache.Cache
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
				config: &robotsHandlerConfig{
					Cache:    boolPtr(true),
					CacheTTL: intPtr(60),
				},
				logger:   log.Default(),
				template: tmpl,
				rwPool:   render.NewRenderWriterPool(),
				cache:    cache.NewCache(),
			},
			args: args{
				w: testRobotsHandlerResponseWriter{},
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
			h := &robotsHandler{
				config:   tt.fields.config,
				logger:   tt.fields.logger,
				template: tt.fields.template,
				rwPool:   tt.fields.rwPool,
				cache:    tt.fields.cache,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
