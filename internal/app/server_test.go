// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"testing"
)

type testServerNextRenderer struct{}

func (r testServerNextRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
}

func (r testServerNextRenderer) Next(renderer Renderer) {
}

func TestCreateServer(t *testing.T) {
	type args struct {
		config    *ServerConfig
		renderers []Renderer
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config:    &ServerConfig{},
				renderers: []Renderer{testServerNextRenderer{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateServer(tt.args.config, tt.args.renderers...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestServerInitialize(t *testing.T) {
	type fields struct {
		config                      *ServerConfig
		logger                      *log.Logger
		reopen                      chan os.Signal
		httpServer                  *http.Server
		renderer                    Renderer
		info                        *ServerInfo
		osReadFile                  func(name string) ([]byte, error)
		httpServerListenAndServe    func(server *http.Server) error
		httpServerListenAndServeTLS func(server *http.Server, certFile string, keyFile string) error
		httpServerShutdown          func(server *http.Server, context context.Context) error
	}
	type args struct {
		renderers []Renderer
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
				config: &ServerConfig{},
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
		},
		{
			name: "access log",
			fields: fields{
				config: &ServerConfig{
					AccessLog:     true,
					AccessLogFile: stringPtr(os.DevNull),
				},
				reopen: make(chan os.Signal, 1),
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
		},
		{
			name: "error create access log file",
			fields: fields{
				config: &ServerConfig{
					AccessLog:     true,
					AccessLogFile: stringPtr(""),
				},
				reopen: make(chan os.Signal, 1),
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
			wantErr: true,
		},
		{
			name: "compress",
			fields: fields{
				config: &ServerConfig{
					Compress: 1,
				},
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
		},
		{
			name: "tls",
			fields: fields{
				config: &ServerConfig{
					TLS:       true,
					TLSCAFile: stringPtr("ca.pem"),
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "ca.pem" {
						return []byte{}, nil
					}
					return nil, errors.New("test error")
				},
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
		},
		{
			name: "tls read file error",
			fields: fields{
				config: &ServerConfig{
					TLS:       true,
					TLSCAFile: stringPtr("ca.pem"),
				},
				osReadFile: func(name string) ([]byte, error) {
					if name == "ca.pem" {
						return nil, errors.New("test error")
					}
					return nil, errors.New("test error")
				},
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:                      tt.fields.config,
				logger:                      tt.fields.logger,
				reopen:                      tt.fields.reopen,
				httpServer:                  tt.fields.httpServer,
				renderer:                    tt.fields.renderer,
				info:                        tt.fields.info,
				osReadFile:                  tt.fields.osReadFile,
				httpServerListenAndServe:    tt.fields.httpServerListenAndServe,
				httpServerListenAndServeTLS: tt.fields.httpServerListenAndServeTLS,
				httpServerShutdown:          tt.fields.httpServerShutdown,
			}
			if err := s.initialize(tt.args.renderers...); (err != nil) != tt.wantErr {
				t.Errorf("server.initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStart(t *testing.T) {
	tlsCertFile := "cert.pem"
	tlsKeyFile := "key.pem"

	type fields struct {
		config                      *ServerConfig
		logger                      *log.Logger
		reopen                      chan os.Signal
		httpServer                  *http.Server
		renderer                    Renderer
		info                        *ServerInfo
		osReadFile                  func(name string) ([]byte, error)
		httpServerListenAndServe    func(server *http.Server) error
		httpServerListenAndServeTLS func(server *http.Server, certFile string, keyFile string) error
		httpServerShutdown          func(server *http.Server, context context.Context) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &ServerConfig{},
				logger: log.Default(),
				httpServer: &http.Server{
					Addr: "localhost:8080",
				},
				httpServerListenAndServe: func(server *http.Server) error {
					return nil
				},
			},
		},
		{
			name: "error listen",
			fields: fields{
				config: &ServerConfig{},
				logger: log.Default(),
				httpServer: &http.Server{
					Addr: "localhost:8080",
				},
				httpServerListenAndServe: func(server *http.Server) error {
					return errors.New("test error")
				},
			},
		},
		{
			name: "tls",
			fields: fields{
				config: &ServerConfig{TLS: true, TLSCertFile: &tlsCertFile, TLSKeyFile: &tlsKeyFile},
				logger: log.Default(),
				httpServer: &http.Server{
					Addr: "localhost:8443",
				},
				httpServerListenAndServeTLS: func(server *http.Server, certFile string, keyFile string) error {
					return nil
				},
			},
		},
		{
			name: "error tls listen",
			fields: fields{
				config: &ServerConfig{TLS: true, TLSCertFile: &tlsCertFile, TLSKeyFile: &tlsKeyFile},
				logger: log.Default(),
				httpServer: &http.Server{
					Addr: "localhost:8443",
				},
				httpServerListenAndServeTLS: func(server *http.Server, certFile string, keyFile string) error {
					return errors.New("test error")
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:                      tt.fields.config,
				logger:                      tt.fields.logger,
				reopen:                      tt.fields.reopen,
				httpServer:                  tt.fields.httpServer,
				renderer:                    tt.fields.renderer,
				info:                        tt.fields.info,
				osReadFile:                  tt.fields.osReadFile,
				httpServerListenAndServe:    tt.fields.httpServerListenAndServe,
				httpServerListenAndServeTLS: tt.fields.httpServerListenAndServeTLS,
				httpServerShutdown:          tt.fields.httpServerShutdown,
			}
			if err := s.Start(); (err != nil) != tt.wantErr {
				t.Errorf("server.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStop(t *testing.T) {
	type fields struct {
		config                      *ServerConfig
		logger                      *log.Logger
		reopen                      chan os.Signal
		httpServer                  *http.Server
		renderer                    Renderer
		info                        *ServerInfo
		osReadFile                  func(name string) ([]byte, error)
		httpServerListenAndServe    func(server *http.Server) error
		httpServerListenAndServeTLS func(server *http.Server, certFile string, keyFile string) error
		httpServerShutdown          func(server *http.Server, context context.Context) error
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
				config: &ServerConfig{},
				logger: log.Default(),
				httpServer: &http.Server{
					Addr: "localhost:8080",
				},
				httpServerShutdown: func(server *http.Server, context context.Context) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
		},
		{
			name: "http server shutdown error",
			fields: fields{
				config: &ServerConfig{},
				logger: log.Default(),
				httpServer: &http.Server{
					Addr: "localhost:8080",
				},
				httpServerShutdown: func(server *http.Server, context context.Context) error {
					return errors.New("test error")
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:                      tt.fields.config,
				logger:                      tt.fields.logger,
				reopen:                      tt.fields.reopen,
				httpServer:                  tt.fields.httpServer,
				renderer:                    tt.fields.renderer,
				info:                        tt.fields.info,
				osReadFile:                  tt.fields.osReadFile,
				httpServerListenAndServe:    tt.fields.httpServerListenAndServe,
				httpServerListenAndServeTLS: tt.fields.httpServerListenAndServeTLS,
				httpServerShutdown:          tt.fields.httpServerShutdown,
			}
			if err := s.Stop(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("server.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testServerHandlerResponseWriter struct {
	http.ResponseWriter
	header http.Header
}

func (w testServerHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testServerHandlerResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testServerHandlerResponseWriter) WriteHeader(statusCode int) {
}

type testServerHandlerRenderer struct{}

func (t *testServerHandlerRenderer) Handle(w http.ResponseWriter, r *http.Request, info *ServerInfo) {
}

func (t *testServerHandlerRenderer) Next(renderer Renderer) {
}

func TestNewServerHandler(t *testing.T) {
	type args struct {
		server *server
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "default",
			args: args{
				server: &server{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewServerHandler(tt.args.server); got == nil {
				t.Errorf("NewServerHandler() = %v", got)
			}
		})
	}
}

func TestServerHandlerServeHTTP(t *testing.T) {
	type fields struct {
		server *server
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
				server: &server{
					renderer: &testServerHandlerRenderer{},
				},
			},
			args: args{
				w: testServerHandlerResponseWriter{
					header: make(http.Header),
				},
				r: &http.Request{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &serverHandler{
				server: tt.fields.server,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
			if tt.args.w.Header().Get("Server") == "" {
				t.Errorf("response header %s is missing", "Server")
			}
			if tt.args.w.Header().Get("X-Request-ID") == "" {
				t.Errorf("response header %s is missing", "X-Request-ID")
			}
		})
	}
}
