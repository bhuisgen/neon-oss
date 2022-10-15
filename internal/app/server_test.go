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
				config: &ServerConfig{},
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
	tlsCAFile := "ca.pem"
	accessLogFile := "access.log"

	serverOsCreateAccessLogFileSuccess := func(name string) (*os.File, error) {
		if name == accessLogFile {
			return nil, nil
		}
		return nil, errors.New("test error")
	}
	serverOsCreateAccessLogFileError := func(name string) (*os.File, error) {
		if name == accessLogFile {
			return nil, errors.New("test error")
		}
		return nil, errors.New("test error")
	}
	serverOsReadFileTLSCAFileSuccess := func(name string) ([]byte, error) {
		if name == tlsCAFile {
			return []byte{}, nil
		}
		return nil, errors.New("test error")
	}
	serverOsReadFileTLSCAFileError := func(name string) ([]byte, error) {
		if name == tlsCAFile {
			return nil, errors.New("test error")
		}
		return nil, errors.New("test error")
	}

	type fields struct {
		config                      *ServerConfig
		logger                      *log.Logger
		httpServer                  *http.Server
		renderer                    Renderer
		info                        *ServerInfo
		osCreate                    func(name string) (*os.File, error)
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
				config:   &ServerConfig{AccessLog: true, AccessLogFile: &accessLogFile},
				osCreate: serverOsCreateAccessLogFileSuccess,
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
		},
		{
			name: "create access log file error",
			fields: fields{
				config:   &ServerConfig{AccessLog: true, AccessLogFile: &accessLogFile},
				osCreate: serverOsCreateAccessLogFileError,
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
			wantErr: true,
		},
		{
			name: "tls",
			fields: fields{
				config:     &ServerConfig{TLS: true, TLSCAFile: &tlsCAFile},
				osReadFile: serverOsReadFileTLSCAFileSuccess,
			},
			args: args{
				renderers: []Renderer{&defaultRenderer{}, &errorRenderer{}},
			},
		},
		{
			name: "tls read file error",
			fields: fields{
				config:     &ServerConfig{TLS: true, TLSCAFile: &tlsCAFile},
				osReadFile: serverOsReadFileTLSCAFileError,
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
				httpServer:                  tt.fields.httpServer,
				renderer:                    tt.fields.renderer,
				info:                        tt.fields.info,
				osCreate:                    tt.fields.osCreate,
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
		httpServer                  *http.Server
		renderer                    Renderer
		info                        *ServerInfo
		osCreate                    func(name string) (*os.File, error)
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
				httpServer:                  tt.fields.httpServer,
				renderer:                    tt.fields.renderer,
				info:                        tt.fields.info,
				osCreate:                    tt.fields.osCreate,
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
		httpServer                  *http.Server
		renderer                    Renderer
		info                        *ServerInfo
		osCreate                    func(name string) (*os.File, error)
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
				httpServer:                  tt.fields.httpServer,
				renderer:                    tt.fields.renderer,
				info:                        tt.fields.info,
				osCreate:                    tt.fields.osCreate,
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
			if tt.args.w.Header().Get("X-Correlation-ID") == "" {
				t.Errorf("response header %s is missing", "X-Correlation-ID")
			}
		})
	}
}
