package app

import (
	"errors"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"rogchap.com/v8go"
)

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
			got := newVM()
			if (got == nil) != tt.wantNil {
				t.Errorf("newVM() got = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestVMClose(t *testing.T) {
	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	type fields struct {
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		serverHandlerObject         *v8go.ObjectTemplate
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
		context                     *v8go.Context
		config                      *vmConfig
		logger                      *slog.Logger
		status                      vmStatus
		data                        *vmData
		v8NewFunctionTemplate       func(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate
		v8ObjectTemplateNewInstance func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error)
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				logger:  slog.Default(),
				isolate: isolate,
				context: context,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				serverHandlerObject:         tt.fields.serverHandlerObject,
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
				context:                     tt.fields.context,
				config:                      tt.fields.config,
				logger:                      tt.fields.logger,
				status:                      tt.fields.status,
				data:                        tt.fields.data,
				v8NewFunctionTemplate:       tt.fields.v8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: tt.fields.v8ObjectTemplateNewInstance,
			}
			v.Close()
		})
	}
}

func TestVMReset(t *testing.T) {
	type fields struct {
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		serverHandlerObject         *v8go.ObjectTemplate
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
		context                     *v8go.Context
		config                      *vmConfig
		logger                      *slog.Logger
		status                      vmStatus
		data                        *vmData
		v8NewFunctionTemplate       func(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate
		v8ObjectTemplateNewInstance func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error)
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "default",
			fields: fields{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				serverHandlerObject:         tt.fields.serverHandlerObject,
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
				context:                     tt.fields.context,
				config:                      tt.fields.config,
				logger:                      tt.fields.logger,
				status:                      tt.fields.status,
				data:                        tt.fields.data,
				v8NewFunctionTemplate:       tt.fields.v8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: tt.fields.v8ObjectTemplateNewInstance,
			}
			v.Reset()
		})
	}
}

func TestVMConfigure(t *testing.T) {
	isolate := v8go.NewIsolate()
	context := v8go.NewContext(isolate)
	defer isolate.Dispose()
	defer context.Close()

	type fields struct {
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		serverHandlerObject         *v8go.ObjectTemplate
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
		context                     *v8go.Context
		config                      *vmConfig
		logger                      *slog.Logger
		status                      vmStatus
		data                        *vmData
		v8NewFunctionTemplate       func(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate
		v8ObjectTemplateNewInstance func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error)
	}
	type args struct {
		config *vmConfig
		logger *slog.Logger
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
				isolate:               isolate,
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          v8go.NewObjectTemplate(isolate),
				serverHandlerObject:   v8go.NewObjectTemplate(isolate),
				serverRequestObject:   v8go.NewObjectTemplate(isolate),
				serverResponseObject:  v8go.NewObjectTemplate(isolate),
				context:               context,
				logger:                slog.Default(),
				status:                vmStatusNew,
				data:                  &vmData{},
				v8NewFunctionTemplate: vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error) {
					if template == nil {
						return nil, errors.New("test error")
					}
					return vmV8ObjectTemplateNewInstance(template, context)
				},
			},
			args: args{
				config: &vmConfig{
					Env:     "test",
					Request: nil,
					State:   nil,
				},
				logger: slog.Default(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				serverHandlerObject:         tt.fields.serverHandlerObject,
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
				context:                     tt.fields.context,
				config:                      tt.fields.config,
				logger:                      tt.fields.logger,
				status:                      tt.fields.status,
				data:                        tt.fields.data,
				v8NewFunctionTemplate:       tt.fields.v8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: tt.fields.v8ObjectTemplateNewInstance,
			}
			if err := v.Configure(tt.args.config, tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("vm.Configure() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVMExecute(t *testing.T) {
	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	type fields struct {
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		serverHandlerObject         *v8go.ObjectTemplate
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
		context                     *v8go.Context
		config                      *vmConfig
		logger                      *slog.Logger
		status                      vmStatus
		data                        *vmData
		v8NewFunctionTemplate       func(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate
		v8ObjectTemplateNewInstance func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error)
	}
	type args struct {
		name    string
		source  string
		timeout time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				serverHandlerObject:         v8go.NewObjectTemplate(isolate),
				serverRequestObject:         v8go.NewObjectTemplate(isolate),
				serverResponseObject:        v8go.NewObjectTemplate(isolate),
				context:                     context,
				logger:                      slog.Default(),
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				name:    "test1",
				source:  `(() => { const test = "test"; })();`,
				timeout: 200 * time.Millisecond,
			},
			want: &vmResult{},
		},
		{
			name: "script error",
			fields: fields{
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				serverHandlerObject:         v8go.NewObjectTemplate(isolate),
				serverRequestObject:         v8go.NewObjectTemplate(isolate),
				serverResponseObject:        v8go.NewObjectTemplate(isolate),
				context:                     context,
				logger:                      slog.Default(),
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				name:    "test2",
				source:  `(() => { error; })();`,
				timeout: 200 * time.Millisecond,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "timeout",
			fields: fields{
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				serverHandlerObject:         v8go.NewObjectTemplate(isolate),
				serverRequestObject:         v8go.NewObjectTemplate(isolate),
				serverResponseObject:        v8go.NewObjectTemplate(isolate),
				context:                     context,
				logger:                      slog.Default(),
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				name:    "test",
				source:  `(() => { for(;;) {} })();`,
				timeout: 200 * time.Millisecond,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				serverHandlerObject:         tt.fields.serverHandlerObject,
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
				context:                     tt.fields.context,
				config:                      tt.fields.config,
				logger:                      tt.fields.logger,
				status:                      tt.fields.status,
				data:                        tt.fields.data,
				v8NewFunctionTemplate:       tt.fields.v8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: tt.fields.v8ObjectTemplateNewInstance,
			}
			got, err := v.Execute(tt.args.name, tt.args.source, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("vm.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vm.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
