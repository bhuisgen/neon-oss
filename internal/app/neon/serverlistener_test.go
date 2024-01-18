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

type testServerListenerRouterResponseWriter struct {
	header http.Header
}

func (w testServerListenerRouterResponseWriter) Header() http.Header {
	return w.header
}

func (w testServerListenerRouterResponseWriter) Write(b []byte) (int, error) {
	return 0, nil
}

func (w testServerListenerRouterResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testServerListenerRouterResponseWriter)(nil)

type testServerListenerHandlerResponseWriter struct {
	header http.Header
}

func (w testServerListenerHandlerResponseWriter) Header() http.Header {
	return w.header
}

func (w testServerListenerHandlerResponseWriter) Write(b []byte) (int, error) {
	return 0, nil
}

func (w testServerListenerHandlerResponseWriter) WriteHeader(statusCode int) {
}

var _ http.ResponseWriter = (*testServerListenerHandlerResponseWriter)(nil)

func TestServerListenerCheck(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
			fields: fields{
				name: "test",
			},
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
				"unregistered listener module 'unknown'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
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

func TestServerListenerLoad(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
			l := &serverListener{
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

func TestServerListenerRegister(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	type args struct {
		descriptor ServerListenerDescriptor
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
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{},
				},
				osClose: func(f *os.File) error {
					return nil
				},
			},
			args: args{
				descriptor: &serverListenerDescriptor{},
			},
		},
		{
			name: "error register",
			fields: fields{
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{
						errRegister: true,
					},
				},
			},
			args: args{
				descriptor: &serverListenerDescriptor{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
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

func TestServerListenerServe(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{},
				},
			},
		},
		{
			name: "error serve",
			fields: fields{
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{
						errServe: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
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

func TestServerListenerShutdown(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{},
				},
			},
		},
		{
			name: "error shutdown",
			fields: fields{
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{
						errShutdown: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
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

func TestServerListenerClose(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{},
				},
			},
		},
		{
			name: "error close",
			fields: fields{
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{
						errClose: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
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

func TestServerListenerRemove(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
				state: &serverListenerState{
					listenerModule: &testServerListenerModule{},
				},
				quit:   make(chan struct{}, 1),
				update: make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
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

func TestServerListenerName(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
			l := &serverListener{
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

func TestServerListenerLink(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	type args struct {
		site ServerSite
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
				state: &serverListenerState{
					sites: map[string]ServerSite{},
				},
				update: make(chan struct{}, 1),
			},
			args: args{
				site: &serverSite{
					name: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Link(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("listener.Link() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerListenerUnlink(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan struct{}
		osClose func(f *os.File) error
	}
	type args struct {
		site ServerSite
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
				state: &serverListenerState{
					sites: map[string]ServerSite{},
				},
				update: make(chan struct{}, 1),
			},
			args: args{
				site: &serverSite{
					name: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
				name:    tt.fields.name,
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Unlink(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("listener.Unlink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerListenerDescriptor(t *testing.T) {
	type fields struct {
		name    string
		config  *serverListenerConfig
		logger  *log.Logger
		state   *serverListenerState
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
				state: &serverListenerState{
					mediator: &serverListenerMediator{},
				},
				update: make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
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

func TestServerListenerMediatorListeners(t *testing.T) {
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
			m := &serverListenerMediator{
				listeners: tt.fields.listeners,
			}
			if got := m.Listeners(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("listenerMediator.Listeners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerListenerMediatorRegisterListener(t *testing.T) {
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
			m := &serverListenerMediator{
				listeners: tt.fields.listeners,
			}
			if err := m.RegisterListener(tt.args.listener); (err != nil) != tt.wantErr {
				t.Errorf("listenerMediator.RegisterListener() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerListenerDescriptorFiles(t *testing.T) {
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
			d := &serverListenerDescriptor{
				files: tt.fields.files,
			}
			d.Files()
		})
	}
}

func TestServerListenerRouterServeHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fail()
	}

	type fields struct {
		listener *serverListener
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
				listener: &serverListener{},
				logger:   log.Default(),
				mux:      http.NewServeMux(),
			},
			args: args{
				w: testServerListenerRouterResponseWriter{
					header: make(http.Header),
				},
				r: req,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListenerRouter{
				listener: tt.fields.listener,
				logger:   tt.fields.logger,
				mux:      tt.fields.mux,
			}
			l.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}

func TestServerListenerHandlerServeHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fail()
	}

	type fields struct {
		listener *serverListener
		logger   *log.Logger
		router   ServerListenerRouter
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
				listener: &serverListener{},
				logger:   log.Default(),
			},
			args: args{
				w: testServerListenerHandlerResponseWriter{
					header: make(http.Header),
				},
				r: req,
			},
		},
		{
			name: "default with router",
			fields: fields{
				listener: &serverListener{},
				logger:   log.Default(),
				router: &serverListenerRouter{
					mux: mux,
				},
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
			h := &serverListenerHandler{
				listener: tt.fields.listener,
				logger:   tt.fields.logger,
				router:   tt.fields.router,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
