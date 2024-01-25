// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
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

func TestServerListenerInit(t *testing.T) {
	type fields struct {
		name    string
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
		osClose func(f *os.File) error
	}
	type args struct {
		config map[string]interface{}
		logger *log.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "minimal",
			fields: fields{
				name:  "default",
				state: &serverListenerState{},
			},
			args: args{
				logger: log.Default(),
				config: map[string]interface{}{
					"test": map[string]interface{}{
						"test": "abc",
					},
				},
			},
		},
		{
			name: "error unregistered module",
			fields: fields{
				name: "default",
			},
			args: args{
				logger: log.Default(),
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
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				quit:    tt.fields.quit,
				update:  tt.fields.update,
				osClose: tt.fields.osClose,
			}
			if err := l.Init(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("listener.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerListenerRegister(t *testing.T) {
	listener, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		name    string
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
			name: "without descriptors",
			fields: fields{
				logger: log.Default(),
				state: &serverListenerState{
					listener: &testServerListenerModule{},
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
			name: "with descriptors",
			fields: fields{
				logger: log.Default(),
				state: &serverListenerState{
					listener: &testServerListenerModule{},
				},
				osClose: func(f *os.File) error {
					return nil
				},
			},
			args: args{
				descriptor: &serverListenerDescriptor{
					files: []*os.File{
						file,
					},
				},
			},
		},
		{
			name: "error register",
			fields: fields{
				logger: log.Default(),
				state: &serverListenerState{
					listener: &testServerListenerModule{
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
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
					listener: &testServerListenerModule{},
				},
			},
		},
		{
			name: "error serve",
			fields: fields{
				state: &serverListenerState{
					listener: &testServerListenerModule{
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
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
					listener: &testServerListenerModule{},
				},
			},
		},
		{
			name: "error shutdown",
			fields: fields{
				state: &serverListenerState{
					listener: &testServerListenerModule{
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
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
					listener: &testServerListenerModule{},
				},
			},
		},
		{
			name: "error close",
			fields: fields{
				state: &serverListenerState{
					listener: &testServerListenerModule{
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
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
					listener: &testServerListenerModule{},
				},
				quit:   make(chan struct{}, 1),
				update: make(chan chan error, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
				name:    tt.fields.name,
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
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
	createUpdateChan := func(errUpdate bool) chan chan error {
		updateChan := make(chan chan error)
		go func() {
			for {
				errChan := <-updateChan
				if errUpdate {
					errChan <- errors.New("test error")
				} else {
					errChan <- nil
				}
			}
		}()
		return updateChan
	}

	type fields struct {
		name    string
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
				update: createUpdateChan(false),
			},
			args: args{
				site: &serverSite{
					name: "test",
				},
			},
		},
		{
			name: "error update",
			fields: fields{
				state: &serverListenerState{
					sites: map[string]ServerSite{},
				},
				update: createUpdateChan(true),
			},
			args: args{
				site: &serverSite{
					name: "test",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
				name:    tt.fields.name,
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
	createUpdateChan := func(errUpdate bool) chan chan error {
		updateChan := make(chan chan error)
		go func() {
			for {
				errChan := <-updateChan
				if errUpdate {
					errChan <- errors.New("test error")
				} else {
					errChan <- nil
				}
			}
		}()
		return updateChan
	}

	type fields struct {
		name    string
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
				update: createUpdateChan(false),
			},
			args: args{
				site: &serverSite{
					name: "test",
				},
			},
		},
		{
			name: "error update",
			fields: fields{
				state: &serverListenerState{
					sites: map[string]ServerSite{},
				},
				update: createUpdateChan(true),
			},
			args: args{
				site: &serverSite{
					name: "test",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
				name:    tt.fields.name,
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
		logger  *log.Logger
		state   *serverListenerState
		quit    chan struct{}
		update  chan chan error
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
				update: make(chan chan error, 1),
			},
		},
		{
			name: "listener not ready",
			fields: fields{
				state:  &serverListenerState{},
				update: make(chan chan error, 1),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListener{
				name:    tt.fields.name,
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
		logger *log.Logger
		mux    *http.ServeMux
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
				logger: log.Default(),
				mux:    http.NewServeMux(),
			},
			args: args{
				w: testServerListenerRouterResponseWriter{
					header: http.Header{},
				},
				r: req,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &serverListenerRouter{
				logger: tt.fields.logger,
				mux:    tt.fields.mux,
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
		logger *log.Logger
		router ServerListenerRouter
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
				logger: log.Default(),
			},
			args: args{
				w: testServerListenerHandlerResponseWriter{
					header: http.Header{},
				},
				r: req,
			},
		},
		{
			name: "default with router",
			fields: fields{
				logger: log.Default(),
				router: &serverListenerRouter{
					mux: mux,
				},
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
			h := &serverListenerHandler{
				logger: tt.fields.logger,
				router: tt.fields.router,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
