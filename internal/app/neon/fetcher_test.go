package neon

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

func TestFetcherInit(t *testing.T) {
	type fields struct {
		config *fetcherConfig
		logger *slog.Logger
		state  *fetcherState
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
			name: "no configuration",
			fields: fields{
				logger: slog.Default(),
				state: &fetcherState{
					providers: map[string]core.FetcherProviderModule{},
				},
			},
		},
		{
			name: "empty configuration",
			fields: fields{
				logger: slog.Default(),
				state: &fetcherState{
					providers: map[string]core.FetcherProviderModule{},
				},
			},
			args: args{
				config: map[string]interface{}{},
			},
		},
		{
			name: "full",
			fields: fields{
				logger: slog.Default(),
				state: &fetcherState{
					providers: map[string]core.FetcherProviderModule{
						"default": testFetcherProviderModule{},
					},
				},
			},
			args: args{
				config: map[string]interface{}{
					"providers": map[string]interface{}{
						"default": map[string]interface{}{
							"test": map[string]interface{}{},
						},
					},
				},
			},
		},
		{
			name: "error unregistered provider module",
			fields: fields{
				logger: slog.Default(),
				state:  &fetcherState{},
			},
			args: args{
				config: map[string]interface{}{
					"providers": map[string]interface{}{
						"default": map[string]interface{}{
							"unknown": map[string]interface{}{},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := f.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("fetcher.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetcherFetch(t *testing.T) {
	type fields struct {
		config *fetcherConfig
		logger *slog.Logger
		state  *fetcherState
	}
	type args struct {
		ctx      context.Context
		name     string
		provider string
		config   map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *core.Resource
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				logger: slog.Default(),
				state: &fetcherState{
					providers: map[string]core.FetcherProviderModule{
						"test": testFetcherProviderModule{},
					},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			want: &core.Resource{
				Data: [][]byte{[]byte("test")},
				TTL:  0,
			},
		},
		{
			name: "error resource not found",
			fields: fields{
				logger: slog.Default(),
				state:  &fetcherState{},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
		{
			name: "error provider not found",
			fields: fields{
				logger: slog.Default(),
				state: &fetcherState{
					providers: map[string]core.FetcherProviderModule{},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
		{
			name: "error module not found",
			fields: fields{
				logger: slog.Default(),
				state: &fetcherState{
					providers: map[string]core.FetcherProviderModule{},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
		{
			name: "error fetch",
			fields: fields{
				logger: slog.Default(),
				state: &fetcherState{
					providers: map[string]core.FetcherProviderModule{
						"test": testFetcherProviderModule{
							errFetch: true,
						},
					},
				},
			},
			args: args{
				ctx:      context.Background(),
				name:     "test",
				provider: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fetcher{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			got, err := f.Fetch(tt.args.ctx, tt.args.name, tt.args.provider, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetcher.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetcher.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}
