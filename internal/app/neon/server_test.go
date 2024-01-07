// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"log"
	"net/http"
	"reflect"
	"testing"
)

type testServerResponseWriter struct {
	header http.Header
}

func (w testServerResponseWriter) Header() http.Header {
	return w.header
}

func (w testServerResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testServerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testServerResponseWriter)(nil)

func TestServerCheck(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				"server: no listener defined",
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
				"server: unregistered middleware module 'unknown'",
				"server: unregistered handler module 'unknown'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerLoad(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
			s := &server{
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

func TestServerRegister(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerStart(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerEnable(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if err := s.Enable(); (err != nil) != tt.wantErr {
				t.Errorf("server.Enable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerDisable(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		fetcher Fetcher
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if err := s.Disable(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("server.Disable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStop(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerRemove(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				fetcher: tt.fields.fetcher,
			}
			if err := s.Remove(); (err != nil) != tt.wantErr {
				t.Errorf("server.Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerName(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
			s := &server{
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

func TestServerListeners(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{
					listeners: []string{"test"},
				},
			},
			want: []string{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerHosts(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{
					hosts: []string{"test"},
				},
			},
			want: []string{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerDefault(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
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
				state: &serverState{
					defaultServer: true,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerMediatorRegisterMiddleware(t *testing.T) {
	type fields struct {
		server             *server
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
				currentRoute: serverRouteDefault,
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
			m := &serverMediator{
				server:             tt.fields.server,
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

func TestServerMediatorRegisterHandler(t *testing.T) {
	type fields struct {
		server             *server
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
				currentRoute: serverRouteDefault,
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
			m := &serverMediator{
				server:             tt.fields.server,
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

func TestServerRouter(t *testing.T) {
	type fields struct {
		name    string
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		fields  fields
		want    ServerRouter
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverState{
					router: &serverRouter{},
				},
			},
		},
		{
			name: "error server not ready",
			fields: fields{
				state: &serverState{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
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

func TestServerHandlerServeHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fail()
	}

	type fields struct {
		server *server
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
				server: &server{},
				logger: log.Default(),
			},
			args: args{
				w: testListenerHandlerResponseWriter{
					header: make(http.Header),
				},
				r: req,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &serverHandler{
				server: tt.fields.server,
				logger: tt.fields.logger,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
