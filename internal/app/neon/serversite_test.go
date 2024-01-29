// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

type testServerSiteResponseWriter struct {
	header http.Header
}

func (w testServerSiteResponseWriter) Header() http.Header {
	return w.header
}

func (w testServerSiteResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testServerSiteResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testServerSiteResponseWriter)(nil)

func TestServerSiteInit(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				name:   "main",
				logger: slog.Default(),
				state: &serverSiteState{
					routesMap: map[string]serverSiteRouteState{},
				},
			},
			args: args{
				config: map[string]interface{}{
					"listeners": []string{"test"},
					"routes": map[string]interface{}{
						"default": map[string]interface{}{
							"middlewares": map[string]interface{}{
								"test": map[string]interface{}{},
							},
							"handler": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
					},
				},
			},
		},
		{
			name: "error no listener",
			fields: fields{
				name:   "main",
				logger: slog.Default(),
				state: &serverSiteState{
					routesMap: map[string]serverSiteRouteState{},
				},
			},
			args: args{
				config: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "error unregistered modules",
			fields: fields{
				name:   "main",
				logger: slog.Default(),
				state: &serverSiteState{
					routesMap: map[string]serverSiteRouteState{},
				},
			},
			args: args{
				config: map[string]interface{}{
					"listeners": []string{"test"},
					"routes": map[string]interface{}{
						"default": map[string]interface{}{
							"middlewares": map[string]interface{}{
								"unknown": map[string]interface{}{},
							},
							"handler": map[string]interface{}{
								"unknown": map[string]interface{}{},
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
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if err := s.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("server.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerSiteRegister(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "without routes",
			fields: fields{
				logger: slog.Default(),
				state:  &serverSiteState{},
			},
		},
		{
			name: "with routes",
			fields: fields{
				logger: slog.Default(),
				state: &serverSiteState{
					routes: []string{"/"},
					routesMap: map[string]serverSiteRouteState{
						"/": {
							middlewares: map[string]core.ServerSiteMiddlewareModule{
								"test": testServerSiteMiddlewareModule{},
							},
							handler: testServerSiteHandlerModule{},
						},
					},
				},
			},
		},
		{
			name: "error register middleware",
			fields: fields{
				logger: slog.Default(),
				state: &serverSiteState{
					routes: []string{"/"},
					routesMap: map[string]serverSiteRouteState{
						"/": {
							middlewares: map[string]core.ServerSiteMiddlewareModule{
								"test": testServerSiteMiddlewareModule{
									errRegister: true,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error register handler",
			fields: fields{
				logger: slog.Default(),
				state: &serverSiteState{
					routes: []string{"/"},
					routesMap: map[string]serverSiteRouteState{
						"/": {
							handler: testServerSiteHandlerModule{
								errRegister: true},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if err := s.Register(); (err != nil) != tt.wantErr {
				t.Errorf("server.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerSiteStart(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "without routes",
			fields: fields{
				logger: slog.Default(),
				state:  &serverSiteState{},
			},
		},
		{
			name: "with routes",
			fields: fields{
				logger: slog.Default(),
				state: &serverSiteState{
					routes: []string{"/"},
					routesMap: map[string]serverSiteRouteState{
						"/": {
							middlewares: map[string]core.ServerSiteMiddlewareModule{
								"test": testServerSiteMiddlewareModule{},
							},
							handler: testServerSiteHandlerModule{},
						},
					},
				},
			},
		},
		{
			name: "error start middleware",
			fields: fields{
				logger: slog.Default(),
				state: &serverSiteState{
					routes: []string{"/"},
					routesMap: map[string]serverSiteRouteState{
						"/": {
							middlewares: map[string]core.ServerSiteMiddlewareModule{
								"test": testServerSiteMiddlewareModule{
									errStart: true,
								},
							},
							handler: testServerSiteHandlerModule{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error start handler",
			fields: fields{
				logger: slog.Default(),
				state: &serverSiteState{
					routes: []string{"/"},
					routesMap: map[string]serverSiteRouteState{
						"/": {
							handler: testServerSiteHandlerModule{
								errStart: true,
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
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if err := s.Start(); (err != nil) != tt.wantErr {
				t.Errorf("server.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerSiteStop(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "without routes",
			fields: fields{
				logger: slog.Default(),
				state:  &serverSiteState{},
			},
		},
		{
			name: "with routes",
			fields: fields{
				logger: slog.Default(),
				state: &serverSiteState{
					routes: []string{"/"},
					routesMap: map[string]serverSiteRouteState{
						"/": {
							middlewares: map[string]core.ServerSiteMiddlewareModule{
								"test": testServerSiteMiddlewareModule{},
							},
							handler: testServerSiteHandlerModule{},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if err := s.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("server.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerSiteName(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "default",
			fields: fields{
				name: "test",
			},
			want: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if got := s.Name(); got != tt.want {
				t.Errorf("server.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerSiteListeners(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "default",
			fields: fields{
				state: &serverSiteState{
					listeners: []string{"test"},
				},
			},
			want: []string{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if got := s.Listeners(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.Listeners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerSiteHosts(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "default",
			fields: fields{
				state: &serverSiteState{
					hosts: []string{"test"},
				},
			},
			want: []string{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if got := s.Hosts(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.Hosts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerSiteDefault(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverSiteState{
					defaultSite: true,
				},
			},
			want: true,
		},
		{
			name: "not default",
			fields: fields{
				state: &serverSiteState{
					defaultSite: false,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if got := s.Default(); got != tt.want {
				t.Errorf("server.Default() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerSiteRouter(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *slog.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		want    ServerSiteRouter
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverSiteState{
					router: &serverSiteRouter{},
				},
			},
		},
		{
			name: "error server not ready",
			fields: fields{
				state: &serverSiteState{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverSite{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			_, err := s.Router()
			if (err != nil) != tt.wantErr {
				t.Errorf("server.Router() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestServerSiteMediatorRegisterMiddleware(t *testing.T) {
	type fields struct {
		site               *serverSite
		currentRoute       string
		defaultMiddlewares []func(http.Handler) http.Handler
		defaultHandler     http.Handler
		routesMiddlewares  map[string][]func(http.Handler) http.Handler
		routesHandler      map[string]http.Handler
	}
	type args struct {
		f func(next http.Handler) http.Handler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default route",
			fields: fields{
				currentRoute: serverSiteRouteDefault,
			},
		},
		{
			name: "custom route without middlewares",
			fields: fields{
				currentRoute: "/custom",
			},
		},
		{
			name: "custom route with middlewares",
			fields: fields{
				currentRoute: "/custom",
				routesMiddlewares: map[string][]func(http.Handler) http.Handler{
					"/custom": {
						func(h http.Handler) http.Handler {
							return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &serverSiteMediator{
				site:               tt.fields.site,
				currentRoute:       tt.fields.currentRoute,
				defaultMiddlewares: tt.fields.defaultMiddlewares,
				defaultHandler:     tt.fields.defaultHandler,
				routesMiddlewares:  tt.fields.routesMiddlewares,
				routesHandler:      tt.fields.routesHandler,
			}
			if err := m.RegisterMiddleware(tt.args.f); (err != nil) != tt.wantErr {
				t.Errorf("serverMediator.RegisterMiddleware() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerSiteMediatorRegisterHandler(t *testing.T) {
	type fields struct {
		site               *serverSite
		currentRoute       string
		defaultMiddlewares []func(http.Handler) http.Handler
		defaultHandler     http.Handler
		routesMiddlewares  map[string][]func(http.Handler) http.Handler
		routesHandler      map[string]http.Handler
	}
	type args struct {
		h http.Handler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default route",
			fields: fields{
				currentRoute: serverSiteRouteDefault,
			},
		},
		{
			name: "custom route without handler",
			fields: fields{
				currentRoute: "/custom",
			},
		},
		{
			name: "custom route with handler",
			fields: fields{
				currentRoute: "/custom",
				routesHandler: map[string]http.Handler{
					"/custom": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &serverSiteMediator{
				site:               tt.fields.site,
				currentRoute:       tt.fields.currentRoute,
				defaultMiddlewares: tt.fields.defaultMiddlewares,
				defaultHandler:     tt.fields.defaultHandler,
				routesMiddlewares:  tt.fields.routesMiddlewares,
				routesHandler:      tt.fields.routesHandler,
			}
			if err := m.RegisterHandler(tt.args.h); (err != nil) != tt.wantErr {
				t.Errorf("serverMediator.RegisterHandler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerSiteRouterRoutes(t *testing.T) {
	type fields struct {
		logger *slog.Logger
		routes map[string]http.Handler
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]http.Handler
	}{
		{
			name: "default",
			fields: fields{
				routes: map[string]http.Handler{},
			},
			want: map[string]http.Handler{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &serverSiteRouter{
				logger: tt.fields.logger,
				routes: tt.fields.routes,
			}
			if got := r.Routes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("serverSiteRouter.Routes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerSiteMiddlewareHandler(t *testing.T) {
	type fields struct {
		logger *slog.Logger
	}
	type args struct {
		next http.Handler
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   http.Handler
	}{
		{
			name: "default",
			fields: fields{
				logger: slog.Default(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &serverSiteMiddleware{
				logger: tt.fields.logger,
			}
			h := m.Handler(tt.args.next)
			w := httptest.NewRecorder()
			r, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			h.ServeHTTP(w, r)
			if v := w.Header().Get(serverSiteMiddlewareHeaderServer); v != serverSiteMiddlewareHeaderServerValue {
				t.Errorf("missing header")
			}
			if v := w.Header().Get(serverSiteMiddlewareHeaderRequestId); v == "" {
				t.Errorf("missing header")
			}
		})
	}
}

func TestServerSiteHandlerServeHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Error(err)
	}

	type fields struct {
		logger *slog.Logger
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
				logger: slog.Default(),
			},
			args: args{
				w: testServerListenerHandlerResponseWriter{
					header: http.Header{},
				},
				r: req,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &serverSiteHandler{
				logger: tt.fields.logger,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
