package file

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	"github.com/bhuisgen/neon/pkg/core"
)

func TestFileProviderInit(t *testing.T) {
	type fields struct {
		config     *fileProviderConfig
		logger     *slog.Logger
		osReadFile func(name string) ([]byte, error)
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
				config: map[string]interface{}{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &fileProvider{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				osReadFile: tt.fields.osReadFile,
			}
			if err := p.Init(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("fileProvider.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileProviderFetch(t *testing.T) {
	type fields struct {
		config     *fileProviderConfig
		logger     *slog.Logger
		osReadFile func(name string) ([]byte, error)
	}
	type args struct {
		ctx    context.Context
		name   string
		config map[string]interface{}
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
				osReadFile: func(name string) ([]byte, error) {
					return []byte("test"), nil
				},
			},
			args: args{
				config: map[string]interface{}{},
			},
			want: &core.Resource{
				Data: [][]byte{[]byte("test")},
				TTL:  0,
			},
		},
		{
			name: "error read file",
			fields: fields{
				osReadFile: func(name string) ([]byte, error) {
					return nil, errors.New("test error")
				},
			},
			args: args{
				config: map[string]interface{}{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fileProvider{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				osReadFile: tt.fields.osReadFile,
			}
			got, err := f.Fetch(tt.args.ctx, tt.args.name, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("fileProvider.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fileProvider.Fetch() = %v, want %v", got, tt.want)
			}
		})
	}
}
