// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"log"
	"net/http"
	"reflect"
	"testing"
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

func TestServerSiteCheck(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *log.Logger
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
		want    []string
		wantErr bool
	}{
		{
			name: "default",
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
			args: args{
				config: map[string]interface{}{},
			},
			want: []string{
				"site: no listener defined",
			},
			wantErr: true,
		},
		{
			name: "error unregistered modules",
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
			want: []string{
				"site: unregistered middleware module 'unknown'",
				"site: unregistered handler module 'unknown'",
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
			got, err := s.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerSiteLoad(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *log.Logger
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
		wantErr bool
	}{
		{
			name: "default",
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
			name: "error unregistered modules",
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
			if err := s.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("server.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerSiteRegister(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *log.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverSiteState{},
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
		logger  *log.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverSiteState{},
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
		logger  *log.Logger
		state   *serverSiteState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverSiteState{},
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
		logger  *log.Logger
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
		logger  *log.Logger
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
		logger  *log.Logger
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
		logger  *log.Logger
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

func TestServerSiteRouter(t *testing.T) {
	type fields struct {
		name    string
		config  *serverSiteConfig
		logger  *log.Logger
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

func TestServerSiteHandlerServeHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fail()
	}

	type fields struct {
		site   *serverSite
		logger *log.Logger
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
				site:   &serverSite{},
				logger: log.Default(),
			},
			args: args{
				w: testServerListenerHandlerResponseWriter{
					header: make(http.Header),
				},
				r: req,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &serverSiteHandler{
				site:   tt.fields.site,
				logger: tt.fields.logger,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
