package js

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/bhuisgen/gomonkey"
)

func TestMain(m *testing.M) {
	gomonkey.Init()
	code := m.Run()
	gomonkey.ShutDown()
	os.Exit(code)
}

func TestNewVM(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newVM()
			if err != nil {
				t.Errorf("newVM() err = %v", err)
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("newVM() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestVMExecute(t *testing.T) {
	type fields struct {
		config *vmConfig
		logger *slog.Logger
		data   *vmData
	}
	type args struct {
		config  vmConfig
		name    string
		code    []byte
		timeout time.Duration
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
				data:   &vmData{},
			},
			args: args{
				config: vmConfig{
					Env: "test",
				},
				name:    "test",
				code:    []byte(`(() => { const test = "test"; })();`),
				timeout: 4 * time.Second,
			},
		},
		{
			name: "script error",
			fields: fields{
				logger: slog.Default(),
				data:   &vmData{},
			},
			args: args{
				config: vmConfig{
					Env: "test",
				},
				name:    "test",
				code:    []byte(`(() => {`),
				timeout: 4 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "timeout",
			fields: fields{
				logger: slog.Default(),
				data:   &vmData{},
			},
			args: args{
				config: vmConfig{
					Env: "test",
				},
				name:    "test",
				code:    []byte(`(() => { for(;;) {} })();`),
				timeout: 10 * time.Millisecond,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				config: tt.fields.config,
				logger: tt.fields.logger,
				data:   tt.fields.data,
			}
			_, err := v.Execute(tt.args.config, tt.args.name, tt.args.code, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("vm.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
