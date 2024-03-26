package neon

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

type testServerServerListener struct {
	name         string
	errInit      bool
	errRegister  bool
	errServe     bool
	errShutdown  bool
	errClose     bool
	errRemove    bool
	errListeners bool
}

func (l testServerServerListener) Init(config map[string]interface{}) error {
	if l.errInit {
		return errors.New("test error")
	}
	return nil
}

func (l testServerServerListener) Register(app core.App) error {
	if l.errRegister {
		return errors.New("test error")
	}
	return nil
}

func (l testServerServerListener) Serve() error {
	if l.errServe {
		return errors.New("test error")
	}
	return nil
}

func (l testServerServerListener) Shutdown(ctx context.Context) error {
	if l.errShutdown {
		return errors.New("test error")
	}
	return nil
}

func (l testServerServerListener) Close() error {
	if l.errClose {
		return errors.New("test error")
	}
	return nil
}

func (l testServerServerListener) Remove() error {
	if l.errRemove {
		return errors.New("test error")
	}
	return nil
}

func (l testServerServerListener) Name() string {
	return l.name
}

func (l testServerServerListener) Link(site ServerSite) error {
	return nil
}

func (l testServerServerListener) Unlink(site ServerSite) error {
	return nil
}

func (l testServerServerListener) Listeners() ([]net.Listener, error) {
	if l.errListeners {
		return nil, errors.New("test error")
	}
	return nil, nil
}

var _ ServerListener = (*testServerServerListener)(nil)

type testServerServerSite struct {
	name        string
	hosts       []string
	errInit     bool
	errRegister bool
	errStart    bool
	errStop     bool
}

func (s testServerServerSite) Init(config map[string]interface{}) error {
	if s.errInit {
		return errors.New("test error")
	}
	return nil
}

func (s testServerServerSite) Register(app core.App) error {
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

var _ ServerSite = (*testServerServerSite)(nil)

func TestServerInit(t *testing.T) {
	type fields struct {
		config *serverConfig
		logger *slog.Logger
		state  *serverState
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
			fields: fields{
				logger: slog.Default(),
				state: &serverState{
					listenersMap: map[string]ServerListener{},
					sitesMap:     map[string]ServerSite{},
				},
			},
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
			fields: fields{
				logger: slog.Default(),
				state: &serverState{
					listenersMap: map[string]ServerListener{},
					sitesMap:     map[string]ServerSite{},
				},
			},
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
			wantErr: true,
		},
		{
			name: "error no site",
			fields: fields{
				logger: slog.Default(),
				state: &serverState{
					listenersMap: map[string]ServerListener{},
					sitesMap:     map[string]ServerSite{},
				},
			},
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
			wantErr: true,
		},
		{
			name: "error check listener",
			fields: fields{
				logger: slog.Default(),
				state: &serverState{
					listenersMap: map[string]ServerListener{},
					sitesMap:     map[string]ServerSite{},
				},
			},
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
			wantErr: true,
		},
		{
			name: "error check site",
			fields: fields{
				logger: slog.Default(),
				state: &serverState{
					listenersMap: map[string]ServerListener{},
					sitesMap:     map[string]ServerSite{},
				},
			},
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
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("server.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerRegister(t *testing.T) {
	type fields struct {
		config *serverConfig
		logger *slog.Logger
		state  *serverState
	}
	type args struct {
		app core.App
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
				logger: slog.Default(),
				state:  &serverState{},
			},
			args: args{
				app: &appMediator{},
			},
		},
		{
			name: "error listener register",
			fields: fields{
				logger: slog.Default(),
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"test": testServerServerListener{
							errRegister: true,
						},
					},
				},
			},
			args: args{
				app: &appMediator{},
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
				logger: slog.Default(),
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
			args: args{
				app: &appMediator{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Register(tt.args.app); (err != nil) != tt.wantErr {
				t.Errorf("server.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStart(t *testing.T) {
	type fields struct {
		config *serverConfig
		logger *slog.Logger
		state  *serverState
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
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Start(); (err != nil) != tt.wantErr {
				t.Errorf("server.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStop(t *testing.T) {
	type fields struct {
		config *serverConfig
		logger *slog.Logger
		state  *serverState
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("server.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerShutdown(t *testing.T) {
	type fields struct {
		config *serverConfig
		logger *slog.Logger
		state  *serverState
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
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				logger: slog.Default(),
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
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Shutdown(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("server.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerMediatorListeners(t *testing.T) {
	type fields struct {
		config *serverConfig
		logger *slog.Logger
		state  *serverState
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
						"default": testServerServerListener{},
					},
				},
			},
		},
		{
			name: "error get listeners",
			fields: fields{
				state: &serverState{
					listenersMap: map[string]ServerListener{
						"default": testServerServerListener{
							errListeners: true,
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
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if _, err := s.Listeners(); (err != nil) != tt.wantErr {
				t.Errorf("server.Listeners() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
