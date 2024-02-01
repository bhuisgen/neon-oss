package neon

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

func TestStoreInit(t *testing.T) {
	type fields struct {
		config *storeConfig
		logger *slog.Logger
		state  *storeState
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
				state:  &storeState{},
			},
		},
		{
			name: "empty configuration",
			fields: fields{
				logger: slog.Default(),
				state:  &storeState{},
			},
			args: args{
				config: map[string]interface{}{
					"storage": map[string]interface{}{
						"test": map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "default",
			fields: fields{
				logger: slog.Default(),
				state:  &storeState{},
			},
			args: args{
				config: map[string]interface{}{
					"storage": map[string]interface{}{
						"test": map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "error no storage",
			fields: fields{
				logger: slog.Default(),
				state:  &storeState{},
			},
			args: args{
				config: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "error unregistered storage module",
			fields: fields{
				logger: slog.Default(),
				state:  &storeState{},
			},
			args: args{
				config: map[string]interface{}{
					"storage": map[string]interface{}{
						"unknown": map[string]interface{}{},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &store{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("store.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStoreLoadResource(t *testing.T) {
	type fields struct {
		config *storeConfig
		logger *slog.Logger
		state  *storeState
	}
	type args struct {
		name string
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
				state: &storeState{
					storage: &testStoreStorageModule{},
				},
			},
			args: args{
				name: "test",
			},
			want: &core.Resource{
				Data: [][]byte{[]byte("test")},
				TTL:  0,
			},
		},
		{
			name: "error module",
			fields: fields{
				logger: slog.Default(),
				state: &storeState{
					storage: &testStoreStorageModule{
						errLoadResource: true,
					},
				},
			},
			args: args{
				name: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &store{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			got, err := s.LoadResource(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("store.LoadResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("store.LoadResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreStoreResource(t *testing.T) {
	type fields struct {
		config *storeConfig
		logger *slog.Logger
		state  *storeState
	}
	type args struct {
		name     string
		resource *core.Resource
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
				state: &storeState{
					storage: &testStoreStorageModule{},
				},
			},
			args: args{
				name: "test",
				resource: &core.Resource{
					Data: [][]byte{[]byte("test")},
					TTL:  0,
				},
			},
		},
		{
			name: "error module",
			fields: fields{
				logger: slog.Default(),
				state: &storeState{
					storage: &testStoreStorageModule{
						errStoreResource: true,
					},
				},
			},
			args: args{
				name: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &store{
				config: tt.fields.config,
				logger: tt.fields.logger,
				state:  tt.fields.state,
			}
			if err := s.StoreResource(tt.args.name, tt.args.resource); (err != nil) != tt.wantErr {
				t.Errorf("store.StoreResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
