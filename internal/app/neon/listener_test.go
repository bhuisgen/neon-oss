// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type testListenerRouterResponseWriter struct {
	header http.Header
}

func (w testListenerRouterResponseWriter) Header() http.Header {
	return w.header
}

func (w testListenerRouterResponseWriter) Write(b []byte) (int, error) {
	return 0, nil
}

func (w testListenerRouterResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testListenerRouterResponseWriter)(nil)

type testListenerHandlerResponseWriter struct {
	header http.Header
}

func (w testListenerHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testListenerHandlerResponseWriter) Write(b []byte) (int, error) {
	return 0, nil
}

func (w testListenerHandlerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testListenerHandlerResponseWriter)(nil)

func TestListenerCheck(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
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
			args: args{
				config: map[string]interface{}{
					"test": map[string]interface{}{},
				},
			},
		},
		{
			name: "error unregistered module",
			fields: fields{
				name: "test",
			},
			args: args{
				config: map[string]interface{}{
					"unknown": map[string]interface{}{},
				},
			},
			want: []string{
				"listener 'test': unregistered listener module 'unknown'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			got, err := l.Check(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("listener.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("listener.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListenerLoad(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
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
			name: "minimal",
			args: args{
				config: map[string]interface{}{
					"test": map[string]interface{}{},
				},
			},
		},
		{
			name: "error unregistered module",
			args: args{
				config: map[string]interface{}{
					"unknown": map[string]interface{}{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("listener.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerRegister(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	type args struct {
		descriptor ListenerDescriptor
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
				state: &listenerState{
					listenerModule: &testListenerModule{},
				},
				osClose: func(f *os.File) error {
					return nil
				},
			},
			args: args{
				descriptor: &listenerDescriptor{},
			},
		},
		{
			name: "error register",
			fields: fields{
				state: &listenerState{
					listenerModule: &testListenerModule{
						errRegister: true,
					},
				},
			},
			args: args{
				descriptor: &listenerDescriptor{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Register(tt.args.descriptor); (err != nil) != tt.wantErr {
				t.Errorf("listener.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerServe(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &listenerState{
					listenerModule: &testListenerModule{},
				},
			},
		},
		{
			name: "error serve",
			fields: fields{
				state: &listenerState{
					listenerModule: &testListenerModule{
						errServe: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Serve(); (err != nil) != tt.wantErr {
				t.Errorf("listener.Serve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerShutdown(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
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
				state: &listenerState{
					listenerModule: &testListenerModule{},
				},
			},
		},
		{
			name: "error shutdown",
			fields: fields{
				state: &listenerState{
					listenerModule: &testListenerModule{
						errShutdown: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Shutdown(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("listener.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerClose(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &listenerState{
					listenerModule: &testListenerModule{},
				},
			},
		},
		{
			name: "error close",
			fields: fields{
				state: &listenerState{
					listenerModule: &testListenerModule{
						errClose: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Close(); (err != nil) != tt.wantErr {
				t.Errorf("listener.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerRemove(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &listenerState{
					listenerModule: &testListenerModule{},
				},
				quit:   make(chan struct{}, 1),
				update: make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Remove(); (err != nil) != tt.wantErr {
				t.Errorf("listener.Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerName(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
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
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if got := l.Name(); got != tt.want {
				t.Errorf("listener.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListenerLink(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	type args struct {
		server Server
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
				state: &listenerState{
					servers: map[string]Server{},
				},
				update: make(chan struct{}, 1),
			},
			args: args{
				server: &server{
					name: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Link(tt.args.server); (err != nil) != tt.wantErr {
				t.Errorf("listener.Link() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerUnlink(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	type args struct {
		server Server
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
				state: &listenerState{
					servers: map[string]Server{},
				},
				update: make(chan struct{}, 1),
			},
			args: args{
				server: &server{
					name: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Unlink(tt.args.server); (err != nil) != tt.wantErr {
				t.Errorf("listener.Unlink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerDescriptor(t *testing.T) {
	type fields struct {
		name    string
		config  *listenerConfig
		logger  *log.Logger
		state   *listenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &listenerState{
					mediator: &listenerMediator{},
				},
				update: make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			_, err := l.Descriptor()
			if (err != nil) != tt.wantErr {
				t.Errorf("listener.Descriptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestListenerMediatorRegisterListener(t *testing.T) {
	type fields struct {
		listeners []net.Listener
	}
	type args struct {
		listener net.Listener
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
				listener: &net.TCPListener{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &listenerMediator{
				listeners: tt.fields.listeners,
			}
			if err := m.RegisterListener(tt.args.listener); (err != nil) != tt.wantErr {
				t.Errorf("listenerMediator.RegisterListener() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenerMediatorListeners(t *testing.T) {
	type fields struct {
		listeners []net.Listener
	}
	tests := []struct {
		name   string
		fields fields
		want   []net.Listener
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &listenerMediator{
				listeners: tt.fields.listeners,
			}
			if got := m.Listeners(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("listenerMediator.Listeners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListenerDescriptorFiles(t *testing.T) {
	type fields struct {
		files []*os.File
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
			d := &listenerDescriptor{
				files: tt.fields.files,
			}
			d.Files()
		})
	}
}

func TestListenerRouterServeHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fail()
	}

	type fields struct {
		listener *listener
		logger   *log.Logger
		mux      *http.ServeMux
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
				listener: &listener{},
				logger:   log.Default(),
				mux:      http.NewServeMux(),
			},
			args: args{
				w: testListenerRouterResponseWriter{
					header: make(http.Header),
				},
				r: req,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listenerRouter{
				listener: tt.fields.listener,
				logger:   tt.fields.logger,
				mux:      tt.fields.mux,
			}
			l.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}

func TestListenerHandlerServeHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fail()
	}

	type fields struct {
		listener *listener
		logger   *log.Logger
		router   ListenerRouter
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
				listener: &listener{},
				logger:   log.Default(),
			},
			args: args{
				w: testListenerHandlerResponseWriter{
					header: make(http.Header),
				},
				r: req,
			},
		},
		{
			name: "default with router",
			fields: fields{
				listener: &listener{},
				logger:   log.Default(),
				router: &listenerRouter{
					mux: mux,
				},
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
			h := &listenerHandler{
				listener: tt.fields.listener,
				logger:   tt.fields.logger,
				router:   tt.fields.router,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
