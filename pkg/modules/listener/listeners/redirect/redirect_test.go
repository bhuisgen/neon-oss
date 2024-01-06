// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package redirect

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

type testRedirectListener struct {
	errRegister bool
}

func (l testRedirectListener) Name() string {
	return "test"
}

func (l testRedirectListener) RegisterListener(listener net.Listener) error {
	if l.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (l testRedirectListener) Listeners() []net.Listener {
	return nil
}

var _ core.Listener = (*testRedirectListener)(nil)

func TestRedirectListenerModuleInfo(t *testing.T) {
	type fields struct {
		config             *redirectListenerConfig
		logger             *log.Logger
		listener           net.Listener
		server             *http.Server
		osReadFile         func(name string) ([]byte, error)
		netListen          func(network string, addr string) (net.Listener, error)
		httpServerServe    func(server *http.Server, listener net.Listener) error
		httpServerShutdown func(server *http.Server, context context.Context) error
		httpServerClose    func(server *http.Server) error
	}
	tests := []struct {
		name   string
		fields fields
		want   module.ModuleInfo
	}{
		{
			name: "default",
			want: module.ModuleInfo{
				ID:          redirectModuleID,
				NewInstance: func() module.Module { return &redirectListener{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := redirectListener{
				config:             tt.fields.config,
				logger:             tt.fields.logger,
				listener:           tt.fields.listener,
				server:             tt.fields.server,
				osReadFile:         tt.fields.osReadFile,
				netListen:          tt.fields.netListen,
				httpServerServe:    tt.fields.httpServerServe,
				httpServerShutdown: tt.fields.httpServerShutdown,
				httpServerClose:    tt.fields.httpServerClose,
			}
			got := l.ModuleInfo()
			if got.ID != tt.want.ID {
				t.Errorf("redirectListener.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("redirectListener.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestRedirectListenerCheck(t *testing.T) {
	type fields struct {
		config             *redirectListenerConfig
		logger             *log.Logger
		listener           net.Listener
		server             *http.Server
		osReadFile         func(name string) ([]byte, error)
		netListen          func(network string, addr string) (net.Listener, error)
		httpServerServe    func(server *http.Server, listener net.Listener) error
		httpServerShutdown func(server *http.Server, context context.Context) error
		httpServerClose    func(server *http.Server) error
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
		},
		{
			name: "full",
			args: args{
				config: map[string]interface{}{
					"ListenAddr":        "0.0.0.0",
					"ListenPort":        8080,
					"ReadTimeout":       30,
					"ReadHeaderTimeout": 4,
					"WriteTimeout":      30,
					"IdleTimeout":       60,
					"RedirectPort":      8443,
				},
			},
		},
		{
			name: "invalid values",
			args: args{
				config: map[string]interface{}{
					"ListenPort":        -1,
					"ReadTimeout":       -1,
					"ReadHeaderTimeout": -1,
					"WriteTimeout":      -1,
					"IdleTimeout":       -1,
					"RedirectPort":      -1,
				},
			},
			want: []string{
				"option 'ListenPort', invalid value '-1'",
				"option 'ReadTimeout', invalid value '-1'",
				"option 'ReadHeaderTimeout', invalid value '-1'",
				"option 'WriteTimeout', invalid value '-1'",
				"option 'IdleTimeout', invalid value '-1'",
				"option 'RedirectPort', invalid value '-1'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &redirectListener{
				config:             tt.fields.config,
				logger:             tt.fields.logger,
				listener:           tt.fields.listener,
				server:             tt.fields.server,
				osReadFile:         tt.fields.osReadFile,
				netListen:          tt.fields.netListen,
				httpServerServe:    tt.fields.httpServerServe,
				httpServerShutdown: tt.fields.httpServerShutdown,
				httpServerClose:    tt.fields.httpServerClose,
			}
			got, err := l.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("redirectListener.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("redirectListener.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedirectListenerLoad(t *testing.T) {
	type fields struct {
		config             *redirectListenerConfig
		logger             *log.Logger
		listener           net.Listener
		server             *http.Server
		osReadFile         func(name string) ([]byte, error)
		netListen          func(network string, addr string) (net.Listener, error)
		httpServerServe    func(server *http.Server, listener net.Listener) error
		httpServerShutdown func(server *http.Server, context context.Context) error
		httpServerClose    func(server *http.Server) error
	}
	type args struct {
		config    map[string]interface{}
		logPrefix string
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
			l := &redirectListener{
				config:             tt.fields.config,
				logger:             tt.fields.logger,
				listener:           tt.fields.listener,
				server:             tt.fields.server,
				osReadFile:         tt.fields.osReadFile,
				netListen:          tt.fields.netListen,
				httpServerServe:    tt.fields.httpServerServe,
				httpServerShutdown: tt.fields.httpServerShutdown,
				httpServerClose:    tt.fields.httpServerClose,
			}
			if err := l.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("redirectListener.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedirectListenerRegister(t *testing.T) {
	type fields struct {
		config             *redirectListenerConfig
		logger             *log.Logger
		listener           net.Listener
		server             *http.Server
		osReadFile         func(name string) ([]byte, error)
		netListen          func(network string, addr string) (net.Listener, error)
		httpServerServe    func(server *http.Server, listener net.Listener) error
		httpServerShutdown func(server *http.Server, context context.Context) error
		httpServerClose    func(server *http.Server) error
	}
	type args struct {
		listener core.Listener
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
				config: &redirectListenerConfig{
					ListenAddr: stringPtr(redirectConfigDefaultListenAddr),
					ListenPort: intPtr(redirectConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, nil
				},
			},
			args: args{
				listener: testRedirectListener{},
			},
		},
		{
			name: "error listen",
			fields: fields{
				config: &redirectListenerConfig{
					ListenAddr: stringPtr(redirectConfigDefaultListenAddr),
					ListenPort: intPtr(redirectConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				listener: testRedirectListener{},
			},
			wantErr: true,
		},
		{
			name: "error register",
			fields: fields{
				config: &redirectListenerConfig{
					ListenAddr: stringPtr(redirectConfigDefaultListenAddr),
					ListenPort: intPtr(redirectConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, nil
				},
			},
			args: args{
				listener: testRedirectListener{
					errRegister: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &redirectListener{
				config:             tt.fields.config,
				logger:             tt.fields.logger,
				listener:           tt.fields.listener,
				server:             tt.fields.server,
				osReadFile:         tt.fields.osReadFile,
				netListen:          tt.fields.netListen,
				httpServerServe:    tt.fields.httpServerServe,
				httpServerShutdown: tt.fields.httpServerShutdown,
				httpServerClose:    tt.fields.httpServerClose,
			}
			if err := l.Register(tt.args.listener); (err != nil) != tt.wantErr {
				t.Errorf("redirectListener.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedirectListenerServe(t *testing.T) {
	type fields struct {
		config             *redirectListenerConfig
		logger             *log.Logger
		listener           net.Listener
		server             *http.Server
		osReadFile         func(name string) ([]byte, error)
		netListen          func(network string, addr string) (net.Listener, error)
		httpServerServe    func(server *http.Server, listener net.Listener) error
		httpServerShutdown func(server *http.Server, context context.Context) error
		httpServerClose    func(server *http.Server) error
	}
	type args struct {
		handler http.Handler
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
				config: &redirectListenerConfig{
					ListenAddr:        stringPtr(redirectConfigDefaultListenAddr),
					ListenPort:        intPtr(redirectConfigDefaultListenPort),
					ReadTimeout:       intPtr(30),
					ReadHeaderTimeout: intPtr(4),
					WriteTimeout:      intPtr(30),
					IdleTimeout:       intPtr(60),
				},
				logger: log.Default(),
				httpServerServe: func(server *http.Server, listener net.Listener) error {
					return nil
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &redirectListener{
				config:             tt.fields.config,
				logger:             tt.fields.logger,
				listener:           tt.fields.listener,
				server:             tt.fields.server,
				osReadFile:         tt.fields.osReadFile,
				netListen:          tt.fields.netListen,
				httpServerServe:    tt.fields.httpServerServe,
				httpServerShutdown: tt.fields.httpServerShutdown,
				httpServerClose:    tt.fields.httpServerClose,
			}
			if err := l.Serve(tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("redirectListener.Serve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedirectListenerShutdown(t *testing.T) {
	type fields struct {
		config             *redirectListenerConfig
		logger             *log.Logger
		listener           net.Listener
		server             *http.Server
		osReadFile         func(name string) ([]byte, error)
		netListen          func(network string, addr string) (net.Listener, error)
		httpServerServe    func(server *http.Server, listener net.Listener) error
		httpServerShutdown func(server *http.Server, context context.Context) error
		httpServerClose    func(server *http.Server) error
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
				httpServerShutdown: func(server *http.Server, context context.Context) error {
					return nil
				},
			},
		},
		{
			name: "error shutdown",
			fields: fields{
				httpServerShutdown: func(server *http.Server, context context.Context) error {
					return errors.New("test error")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &redirectListener{
				config:             tt.fields.config,
				logger:             tt.fields.logger,
				listener:           tt.fields.listener,
				server:             tt.fields.server,
				osReadFile:         tt.fields.osReadFile,
				netListen:          tt.fields.netListen,
				httpServerServe:    tt.fields.httpServerServe,
				httpServerShutdown: tt.fields.httpServerShutdown,
				httpServerClose:    tt.fields.httpServerClose,
			}
			if err := l.Shutdown(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("redirectListener.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedirectListenerClose(t *testing.T) {
	type fields struct {
		config             *redirectListenerConfig
		logger             *log.Logger
		listener           net.Listener
		server             *http.Server
		osReadFile         func(name string) ([]byte, error)
		netListen          func(network string, addr string) (net.Listener, error)
		httpServerServe    func(server *http.Server, listener net.Listener) error
		httpServerShutdown func(server *http.Server, context context.Context) error
		httpServerClose    func(server *http.Server) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				httpServerClose: func(server *http.Server) error {
					return nil
				},
			},
		},
		{
			name: "error close",
			fields: fields{
				httpServerClose: func(server *http.Server) error {
					return errors.New("test error")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &redirectListener{
				config:             tt.fields.config,
				logger:             tt.fields.logger,
				listener:           tt.fields.listener,
				server:             tt.fields.server,
				osReadFile:         tt.fields.osReadFile,
				netListen:          tt.fields.netListen,
				httpServerServe:    tt.fields.httpServerServe,
				httpServerShutdown: tt.fields.httpServerShutdown,
				httpServerClose:    tt.fields.httpServerClose,
			}
			if err := l.Close(); (err != nil) != tt.wantErr {
				t.Errorf("redirectListener.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
