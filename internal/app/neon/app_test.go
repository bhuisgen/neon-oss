package neon

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"
)

func TestAppInit(t *testing.T) {
	type fields struct {
		config *appConfig
		logger *slog.Logger
		state  *appState
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
				state:  &appState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := a.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("app.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppCheck(t *testing.T) {
	type fields struct {
		config *appConfig
		logger *slog.Logger
		state  *appState
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				config: &appConfig{
					Server: map[string]interface{}{
						"listeners": map[string]interface{}{
							"default": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
						"sites": map[string]interface{}{
							"main": map[string]interface{}{
								"listeners": []string{"default"},
							},
						},
					},
				},
				state: &appState{
					store: &store{
						logger: slog.Default(),
						state:  &storeState{},
					},
					fetcher: &fetcher{
						logger: slog.Default(),
						state:  &fetcherState{},
					},
					loader: &loader{
						logger: slog.Default(),
						state:  &loaderState{},
					},
					server: &server{
						logger: slog.Default(),
						state: &serverState{
							listenersMap: map[string]ServerListener{},
							sitesMap:     map[string]ServerSite{},
						},
					},
				},
			},
		},
		{
			name: "error server",
			fields: fields{
				config: &appConfig{
					Server: map[string]interface{}{},
				},
				state: &appState{
					store: &store{
						logger: slog.Default(),
						state:  &storeState{},
					},
					fetcher: &fetcher{
						logger: slog.Default(),
						state:  &fetcherState{},
					},
					loader: &loader{
						logger: slog.Default(),
						state:  &loaderState{},
					},
					server: &server{
						logger: slog.Default(),
						state:  &serverState{},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := a.Check(); (err != nil) != tt.wantErr {
				t.Errorf("app.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppServe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	type fields struct {
		config *appConfig
		logger *slog.Logger
		state  *appState
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
				config: &appConfig{
					Server: map[string]interface{}{
						"listeners": map[string]interface{}{
							"default": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
						"sites": map[string]interface{}{
							"main": map[string]interface{}{
								"listeners": []string{"default"},
							},
						},
					},
				},
				logger: slog.Default(),
				state: &appState{
					store: &store{
						logger: slog.Default(),
						state:  &storeState{},
					},
					fetcher: &fetcher{
						logger: slog.Default(),
						state:  &fetcherState{},
					},
					loader: &loader{
						logger: slog.Default(),
						state:  &loaderState{},
						mu:     &sync.RWMutex{},
					},
					server: &server{
						logger: slog.Default(),
						state: &serverState{
							listenersMap:   map[string]ServerListener{},
							sitesMap:       map[string]ServerSite{},
							sitesListeners: map[string][]ServerListener{},
						},
					},
					mediator: &appMediator{
						app: &app{
							state: &appState{
								store:   &store{},
								fetcher: &fetcher{},
								loader:  &loader{},
								server:  &server{},
							},
						},
					},
				},
			},
			args: args{
				ctx: ctx,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := a.Serve(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("app.Serve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppMediatorRegisterListeners(t *testing.T) {
	type fields struct {
		app *app
	}
	type args struct {
		listeners map[string][]net.Listener
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
				app: &app{
					state: &appState{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &appMediator{
				app: tt.fields.app,
			}
			if err := m.RegisterListeners(tt.args.listeners); (err != nil) != tt.wantErr {
				t.Errorf("appMediator.RegisterListeners() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
