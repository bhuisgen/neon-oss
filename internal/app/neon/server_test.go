// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"log"
	"reflect"
	"testing"
)

type testServerServerListener struct {
	name        string
	errCheck    bool
	errLoad     bool
	errRegister bool
	errServe    bool
	errShutdown bool
	errClose    bool
	errRemove   bool
}

func (l testServerServerListener) Check(config map[string]interface{}) ([]string, error) {
	if l.errCheck {
		return []string{"test error"}, errors.New("test error")
	}
	return nil, nil
}

func (l testServerServerListener) Load(config map[string]interface{}) error {
	if l.errLoad {
		return errors.New("test error")
	}
	return nil
}

// Register implements ServerListener.
func (l testServerServerListener) Register(descriptor ServerListenerDescriptor) error {
	if l.errRegister {
		return errors.New("test error")
	}
	return nil
}

// Serve implements ServerListener.
func (l testServerServerListener) Serve() error {
	if l.errServe {
		return errors.New("test error")
	}
	return nil
}

// Shutdown implements ServerListener.
func (l testServerServerListener) Shutdown(ctx context.Context) error {
	if l.errShutdown {
		return errors.New("test error")
	}
	return nil
}

// Close implements ServerListener.
func (l testServerServerListener) Close() error {
	if l.errClose {
		return errors.New("test error")
	}
	return nil
}

// Remove implements ServerListener.
func (l testServerServerListener) Remove() error {
	if l.errRemove {
		return errors.New("test error")
	}
	return nil
}

// Name implements ServerListener.
func (l testServerServerListener) Name() string {
	return l.name
}

// Link implements ServerListener.
func (l testServerServerListener) Link(site ServerSite) error {
	return nil
}

// Unlink implements ServerListener.
func (l testServerServerListener) Unlink(site ServerSite) error {
	return nil
}

// Descriptor implements ServerListener.
func (l testServerServerListener) Descriptor() (ServerListenerDescriptor, error) {
	return nil, nil
}

var _ (ServerListener) = (*testServerServerListener)(nil)

type testServerServerSite struct {
	name        string
	hosts       []string
	errCheck    bool
	errLoad     bool
	errRegister bool
	errStart    bool
	errStop     bool
}

func (s testServerServerSite) Check(config map[string]interface{}) ([]string, error) {
	if s.errCheck {
		return []string{"test error"}, errors.New("test error")
	}
	return nil, nil
}

func (s testServerServerSite) Load(config map[string]interface{}) error {
	if s.errLoad {
		return errors.New("test error")
	}
	return nil
}

func (s testServerServerSite) Register() error {
	if s.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (s testServerServerSite) Start() error {
	if s.errStart {
		return errors.New("test error")
	}
	return nil
}

func (s testServerServerSite) Stop() error {
	if s.errStop {
		return errors.New("test error")
	}
	return nil
}

func (s testServerServerSite) Name() string {
	return s.name
}

func (s testServerServerSite) Listeners() []string {
	return []string{"default"}
}

func (s testServerServerSite) Hosts() []string {
	return s.hosts
}

func (s testServerServerSite) Router() (ServerSiteRouter, error) {
	return nil, nil
}

var _ (ServerSite) = (*testServerServerSite)(nil)

func TestServerCheck(t *testing.T) {
	type fields struct {
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		store   Store
		fetcher Fetcher
		loader  Loader
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
					"listeners": map[string]interface{}{
						"default": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
					"sites": map[string]interface{}{
						"main": map[string]interface{}{
							"listeners": []string{"default"},
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
			},
		},
		{
			name: "error no listener",
			args: args{
				config: map[string]interface{}{
					"listeners": map[string]interface{}{},
					"sites": map[string]interface{}{
						"main": map[string]interface{}{
							"listeners": []string{"default"},
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
			},
			want: []string{
				"server: no listener defined",
			},
			wantErr: true,
		},
		{
			name: "error no site",
			args: args{
				config: map[string]interface{}{
					"listeners": map[string]interface{}{
						"default": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
					"sites": map[string]interface{}{},
				},
			},
			want: []string{
				"server: no site defined",
			},
			wantErr: true,
		},
		{
			name: "error check listener",
			args: args{
				config: map[string]interface{}{
					"listeners": map[string]interface{}{
						"default": map[string]interface{}{
							"unknown": map[string]interface{}{},
						},
					},
					"sites": map[string]interface{}{
						"main": map[string]interface{}{
							"listeners": []string{"default"},
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
			},
			want: []string{
				"server: listener 'default', failed to check configuration: unregistered listener module 'unknown'",
			},
			wantErr: true,
		},
		{
			name: "error check site",
			args: args{
				config: map[string]interface{}{
					"listeners": map[string]interface{}{
						"default": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
					"sites": map[string]interface{}{
						"main": map[string]interface{}{
							"listeners": []string{"default"},
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
				},
			},
			want: []string{
				"server: site 'main', failed to check configuration: site: unregistered middleware module 'unknown'",
				"server: site 'main', failed to check configuration: site: unregistered handler module 'unknown'",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				store:   tt.fields.store,
				fetcher: tt.fields.fetcher,
				loader:  tt.fields.loader,
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
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		store   Store
		fetcher Fetcher
		loader  Loader
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
					"listeners": map[string]interface{}{
						"default": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
					"sites": map[string]interface{}{
						"main": map[string]interface{}{
							"listeners": []string{"default"},
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
			},
		},
		{
			name: "error load listener",
			args: args{
				config: map[string]interface{}{
					"listeners": map[string]interface{}{
						"default": map[string]interface{}{
							"unknown": map[string]interface{}{},
						},
					},
					"sites": map[string]interface{}{
						"main": map[string]interface{}{
							"listeners": []string{"unknown"},
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
			},
			wantErr: true,
		},
		{
			name: "error load site",
			args: args{
				config: map[string]interface{}{
					"listeners": map[string]interface{}{
						"default": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
					"sites": map[string]interface{}{
						"main": map[string]interface{}{
							"listeners": []string{"default"},
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
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				store:   tt.fields.store,
				fetcher: tt.fields.fetcher,
				loader:  tt.fields.loader,
			}
			if err := s.Load(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("server.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerRegister(t *testing.T) {
	type fields struct {
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		store   Store
		fetcher Fetcher
		loader  Loader
	}
	type args struct {
		descriptors map[string]ServerListenerDescriptor
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "no descriptors",
			fields: fields{
				state: &serverState{},
			},
			args: args{
				descriptors: nil,
			},
		},
		{
			name: "descriptors",
			fields: fields{
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{},
					},
				},
			},
			args: args{
				descriptors: map[string]ServerListenerDescriptor{},
			},
		},
		{
			name: "error listener register",
			fields: fields{
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{
							errRegister: true,
						},
					},
				},
			},
			args: args{
				descriptors: map[string]ServerListenerDescriptor{},
			},
			wantErr: true,
		},
		{
			name: "error register site",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"default": testServerServerListener{},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{
							errRegister: true,
						},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				store:   tt.fields.store,
				fetcher: tt.fields.fetcher,
				loader:  tt.fields.loader,
			}
			if err := s.Register(tt.args.descriptors); (err != nil) != tt.wantErr {
				t.Errorf("server.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStart(t *testing.T) {
	type fields struct {
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		store   Store
		fetcher Fetcher
		loader  Loader
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"default": testServerServerListener{},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
		},
		{
			name: "error serve listener",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"default": testServerServerListener{
							errServe: true,
						},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error start site",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"default": testServerServerListener{},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{
							errStart: true,
						},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				store:   tt.fields.store,
				fetcher: tt.fields.fetcher,
				loader:  tt.fields.loader,
			}
			if err := s.Start(); (err != nil) != tt.wantErr {
				t.Errorf("server.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStop(t *testing.T) {
	type fields struct {
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		store   Store
		fetcher Fetcher
		loader  Loader
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{},
					},
					sitesMap: map[string]ServerSite{
						"test": testServerServerSite{},
					},
				},
			},
		},
		{
			name: "error stop site",
			fields: fields{
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{},
					},
					sitesMap: map[string]ServerSite{
						"test": testServerServerSite{
							errStop: true,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error close listener",
			fields: fields{
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{
							errClose: true,
						},
					},
					sitesMap: map[string]ServerSite{
						"test": testServerServerSite{},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				store:   tt.fields.store,
				fetcher: tt.fields.fetcher,
				loader:  tt.fields.loader,
			}
			if err := s.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("server.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerShutdown(t *testing.T) {
	type fields struct {
		config  *serverConfig
		logger  *log.Logger
		state   *serverState
		store   Store
		fetcher Fetcher
		loader  Loader
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
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{
							name: "main",
						},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
		},
		{
			name: "error shutdown listener",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{
							errShutdown: true,
						},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{
							name: "main",
						},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error stop site",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{
							name:    "main",
							errStop: true,
						},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error close listener",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{
							errClose: true,
						},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{
							name: "main",
						},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error remove listener",
			fields: fields{
				config: &serverConfig{
					Listeners: map[string]map[string]interface{}{
						"default": {
							"test": map[string]interface{}{},
						},
					},
					Sites: map[string]map[string]interface{}{
						"main": {
							"listeners": []string{"default"},
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
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{
							errRemove: true,
						},
					},
					sitesMap: map[string]ServerSite{
						"main": testServerServerSite{
							name: "main",
						},
					},
					sitesListeners: map[string][]ServerListener{
						"main": {&testServerServerListener{}},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				store:   tt.fields.store,
				fetcher: tt.fields.fetcher,
				loader:  tt.fields.loader,
			}
			if err := s.Shutdown(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("server.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
