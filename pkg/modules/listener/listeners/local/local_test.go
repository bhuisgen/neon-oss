// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package local

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

type testLocalListener struct {
	errRegister bool
}

func (l testLocalListener) Name() string {
	return "test"
}

func (l testLocalListener) RegisterListener(listener net.Listener) error {
	if l.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (l testLocalListener) Listeners() []net.Listener {
	return nil
}

var _ core.Listener = (*testLocalListener)(nil)

func TestLocalListenerModuleInfo(t *testing.T) {
	type fields struct {
		config             *localListenerConfig
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
				ID:          localModuleID,
				NewInstance: func() module.Module { return &localListener{} },
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := localListener{
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
				t.Errorf("localListener.ModuleInfo() = %v, want %v", got.ID, tt.want.ID)
			}
			if instance := got.NewInstance(); instance == nil {
				t.Errorf("localListener.NewInstance() = %v, want %v", instance, "not nil")
			}
		})
	}
}

func TestLocalListenerCheck(t *testing.T) {
	type fields struct {
		config             *localListenerConfig
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
			args: args{},
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
					"IdleTimeout":       30,
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
				},
			},
			want: []string{
				"option 'ListenPort', invalid value '-1'",
				"option 'ReadTimeout', invalid value '-1'",
				"option 'ReadHeaderTimeout', invalid value '-1'",
				"option 'WriteTimeout', invalid value '-1'",
				"option 'IdleTimeout', invalid value '-1'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &localListener{
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
				t.Errorf("localListener.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("localListener.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalListenerLoad(t *testing.T) {
	type fields struct {
		config             *localListenerConfig
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
		wantErr bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &localListener{
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
				t.Errorf("localListener.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLocalListenerRegister(t *testing.T) {
	type fields struct {
		config             *localListenerConfig
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
				config: &localListenerConfig{
					ListenAddr: stringPtr(localConfigDefaultListenAddr),
					ListenPort: intPtr(localConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, nil
				},
			},
			args: args{
				listener: testLocalListener{},
			},
		},
		{
			name: "error listen",
			fields: fields{
				config: &localListenerConfig{
					ListenAddr: stringPtr(localConfigDefaultListenAddr),
					ListenPort: intPtr(localConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				listener: testLocalListener{},
			},
			wantErr: true,
		},
		{
			name: "error register",
			fields: fields{
				config: &localListenerConfig{
					ListenAddr: stringPtr(localConfigDefaultListenAddr),
					ListenPort: intPtr(localConfigDefaultListenPort),
				},
				netListen: func(network, addr string) (net.Listener, error) {
					return nil, nil
				},
			},
			args: args{
				listener: testLocalListener{
					errRegister: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &localListener{
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
				t.Errorf("localListener.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLocalListenerServe(t *testing.T) {
	type fields struct {
		config             *localListenerConfig
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
				config: &localListenerConfig{
					ListenAddr:        stringPtr(localConfigDefaultListenAddr),
					ListenPort:        intPtr(localConfigDefaultListenPort),
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
			l := &localListener{
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
				t.Errorf("localListener.Serve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLocalListenerShutdown(t *testing.T) {
	type fields struct {
		config             *localListenerConfig
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
			l := &localListener{
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
				t.Errorf("localListener.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLocalListenerClose(t *testing.T) {
	type fields struct {
		config             *localListenerConfig
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
			l := &localListener{
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
				t.Errorf("localListener.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
