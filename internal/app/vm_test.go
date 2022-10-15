// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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
		logger                      *log.Logger
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		requestObject               *v8go.ObjectTemplate
		responseObject              *v8go.ObjectTemplate
		context                     *v8go.Context
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
				logger:  log.Default(),
				isolate: isolate,
				context: context,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				logger:                      tt.fields.logger,
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				requestObject:               tt.fields.requestObject,
				responseObject:              tt.fields.responseObject,
				context:                     tt.fields.context,
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
		logger                      *log.Logger
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		requestObject               *v8go.ObjectTemplate
		responseObject              *v8go.ObjectTemplate
		context                     *v8go.Context
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
				logger:                      tt.fields.logger,
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				requestObject:               tt.fields.requestObject,
				responseObject:              tt.fields.responseObject,
				context:                     tt.fields.context,
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

	var noState *string = nil
	var state string = `{"loading": false, "error": "", response: "{"key": "value"}"}`

	type fields struct {
		logger                      *log.Logger
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		requestObject               *v8go.ObjectTemplate
		responseObject              *v8go.ObjectTemplate
		context                     *v8go.Context
		data                        *vmData
		v8NewFunctionTemplate       func(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate
		v8ObjectTemplateNewInstance func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error)
	}
	type args struct {
		envName string
		info    *ServerInfo
		req     *http.Request
		state   *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "without server info",
			fields: fields{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				envName: "test",
				info:    nil,
				req:     &http.Request{},
				state:   &state,
			},
			wantErr: true,
		},
		{
			name: "without request",
			fields: fields{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				envName: "test",
				info:    &ServerInfo{},
				req:     nil,
				state:   &state,
			},
			wantErr: true,
		},
		{
			name: "without state",
			fields: fields{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				envName: "test",
				info:    &ServerInfo{},
				req:     &http.Request{},
				state:   noState,
			},
		},
		{
			name: "with state",
			fields: fields{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				envName: "test",
				info:    &ServerInfo{},
				req:     &http.Request{},
				state:   &state,
			},
		},
		{
			name: "invalid process object template",
			fields: fields{
				logger:                log.Default(),
				isolate:               isolate,
				processObject:         nil,
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          v8go.NewObjectTemplate(isolate),
				requestObject:         v8go.NewObjectTemplate(isolate),
				responseObject:        v8go.NewObjectTemplate(isolate),
				context:               context,
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
				envName: "test",
				info:    &ServerInfo{},
				req:     &http.Request{},
				state:   nil,
			},
			wantErr: true,
		},
		{
			name: "invalid env object template",
			fields: fields{
				logger:                log.Default(),
				isolate:               isolate,
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             nil,
				serverObject:          v8go.NewObjectTemplate(isolate),
				requestObject:         v8go.NewObjectTemplate(isolate),
				responseObject:        v8go.NewObjectTemplate(isolate),
				context:               context,
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
				envName: "test",
				info:    &ServerInfo{},
				req:     &http.Request{},
				state:   nil,
			},
			wantErr: true,
		},
		{
			name: "invalid server object template",
			fields: fields{
				logger:                log.Default(),
				isolate:               isolate,
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          nil,
				requestObject:         v8go.NewObjectTemplate(isolate),
				responseObject:        v8go.NewObjectTemplate(isolate),
				context:               context,
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
				envName: "test",
				info:    &ServerInfo{},
				req:     &http.Request{},
				state:   nil,
			},
			wantErr: true,
		},
		{
			name: "invalid request object template",
			fields: fields{
				logger:                log.Default(),
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          v8go.NewObjectTemplate(isolate),
				requestObject:         nil,
				responseObject:        v8go.NewObjectTemplate(isolate),
				context:               context,
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
				envName: "test",
				info:    &ServerInfo{},
				req:     &http.Request{},
				state:   nil,
			},
			wantErr: true,
		},
		{
			name: "invalid response object template",
			fields: fields{
				logger:                log.Default(),
				isolate:               isolate,
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          v8go.NewObjectTemplate(isolate),
				requestObject:         v8go.NewObjectTemplate(isolate),
				responseObject:        nil,
				context:               context,
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
				envName: "test",
				info:    &ServerInfo{},
				req:     &http.Request{},
				state:   nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				logger:                      tt.fields.logger,
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				requestObject:               tt.fields.requestObject,
				responseObject:              tt.fields.responseObject,
				context:                     tt.fields.context,
				data:                        tt.fields.data,
				v8NewFunctionTemplate:       tt.fields.v8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: tt.fields.v8ObjectTemplateNewInstance,
			}
			if err := v.Configure(tt.args.envName, tt.args.info, tt.args.req, tt.args.state); (err != nil) != tt.wantErr {
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
		logger                      *log.Logger
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		requestObject               *v8go.ObjectTemplate
		responseObject              *v8go.ObjectTemplate
		context                     *v8go.Context
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
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				name:    "test1",
				source:  `(() => { const test = "test"; })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "script error",
			fields: fields{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				name:    "test2",
				source:  `(() => { error; })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "timeout",
			fields: fields{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				name:    "test",
				source:  `(() => { for(;;) {} })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				logger:                      tt.fields.logger,
				isolate:                     tt.fields.isolate,
				processObject:               tt.fields.processObject,
				envObject:                   tt.fields.envObject,
				serverObject:                tt.fields.serverObject,
				requestObject:               tt.fields.requestObject,
				responseObject:              tt.fields.responseObject,
				context:                     tt.fields.context,
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

func TestVMExecute_Debug(t *testing.T) {
	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()
	DEBUG = true
	defer func() { DEBUG = false }()

	type fields struct {
		logger                      *log.Logger
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		requestObject               *v8go.ObjectTemplate
		responseObject              *v8go.ObjectTemplate
		context                     *v8go.Context
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
			name: "script error",
			fields: fields{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        &vmData{},
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
			},
			args: args{
				name:    "test",
				source:  `(() => { error`,
				timeout: time.Duration(1) * time.Second,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vm{
				logger:                      log.Default(),
				isolate:                     isolate,
				processObject:               v8go.NewObjectTemplate(isolate),
				envObject:                   v8go.NewObjectTemplate(isolate),
				serverObject:                v8go.NewObjectTemplate(isolate),
				requestObject:               v8go.NewObjectTemplate(isolate),
				responseObject:              v8go.NewObjectTemplate(isolate),
				context:                     context,
				data:                        tt.fields.data,
				v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
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

func TestNewVMResult(t *testing.T) {
	render := []byte{'t', 'e', 's', 't'}
	status := http.StatusOK
	redirect := true
	redirectURL := "http://redirect"
	redirectStatus := http.StatusFound
	headers := map[string]string{
		"key": "value",
	}
	title := "test"
	metas := map[string]map[string]string{
		"test": {
			"key": "value",
		},
	}
	links := map[string]map[string]string{
		"test": {
			"key": "value",
		},
	}
	scripts := map[string]map[string]string{
		"test": {
			"key": "value",
		},
	}

	type args struct {
		d *vmData
	}
	tests := []struct {
		name string
		args args
		want *vmResult
	}{
		{
			name: "empty",
			args: args{
				d: &vmData{
					render:         nil,
					status:         nil,
					redirect:       nil,
					redirectURL:    nil,
					redirectStatus: nil,
					title:          nil,
					headers:        nil,
					metas:          nil,
					links:          nil,
					scripts:        nil,
				},
			},
			want: &vmResult{},
		},
		{
			name: "default",
			args: args{
				d: &vmData{
					render:         &render,
					status:         &status,
					redirect:       &redirect,
					redirectURL:    &redirectURL,
					redirectStatus: &redirectStatus,
					headers:        headers,
					title:          &title,
					metas:          metas,
					links:          links,
					scripts:        scripts,
				},
			},
			want: &vmResult{
				Render:         render,
				Status:         status,
				Redirect:       redirect,
				RedirectURL:    redirectURL,
				RedirectStatus: redirectStatus,
				Headers:        headers,
				Title:          title,
				Metas:          metas,
				Links:          links,
				Scripts:        scripts,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newVMResult(tt.args.d); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newVMResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVM_APIServer(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	info := &ServerInfo{
		Addr:    configDefaultServerListenAddr,
		Port:    configDefaultServerListenPort,
		Version: Version,
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d", info.Addr, info.Port), nil)
	if err != nil {
		t.Errorf("failed create request: %s", err)
	}

	type args struct {
		name    string
		source  string
		timeout time.Duration
		info    *ServerInfo
		req     *http.Request
		state   *string
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "addr",
			args: args{
				name:    "test",
				source:  `(() => { server.addr(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "port",
			args: args{
				name:    "test",
				source:  `(() => { server.port(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "version",
			args: args{
				name:    "test",
				source:  `(() => { server.version(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			err := v.Configure("test", tt.args.info, tt.args.req, tt.args.state)
			if err != nil {
				t.Errorf("failed to configure VM: %s", err)
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

func TestVM_APIRequest(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	info := &ServerInfo{
		Addr:    configDefaultServerListenAddr,
		Port:    configDefaultServerListenPort,
		Version: Version,
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d", info.Addr, info.Port), nil)
	if err != nil {
		t.Errorf("failed create request: %s", err)
	}

	type args struct {
		name    string
		source  string
		timeout time.Duration
		info    *ServerInfo
		req     *http.Request
		state   *string
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "method",
			args: args{
				name:    "test",
				source:  `(() => { request.method(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "proto",
			args: args{
				name:    "test",
				source:  `(() => { request.proto(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "proto major",
			args: args{
				name:    "test",
				source:  `(() => { request.protoMajor(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "proto minor",
			args: args{
				name:    "test",
				source:  `(() => { request.protoMinor(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "remote addr",
			args: args{
				name:    "test",
				source:  `(() => { request.remoteAddr(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "host",
			args: args{
				name:    "test",
				source:  `(() => { request.host(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "path",
			args: args{
				name:    "test",
				source:  `(() => { request.path(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "query",
			args: args{
				name:    "test",
				source:  `(() => { request.query(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "header",
			args: args{
				name:    "test",
				source:  `(() => { request.headers(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
		{
			name: "state",
			args: args{
				name:    "test",
				source:  `(() => { request.state(); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			err = v.Configure("test", tt.args.info, tt.args.req, tt.args.state)
			if err != nil {
				t.Errorf("failed to configure VM: %s", err)
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

func TestVM_APIResponse(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	info := &ServerInfo{
		Addr:    configDefaultServerListenAddr,
		Port:    configDefaultServerListenPort,
		Version: Version,
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d", info.Addr, info.Port), nil)
	if err != nil {
		t.Errorf("failed create request: %s", err)
	}

	type args struct {
		name    string
		source  string
		timeout time.Duration
		info    *ServerInfo
		req     *http.Request
		state   *string
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "render without status code",
			args: args{
				name:    "test",
				source:  `(() => { response.render("test"); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Render: []byte("test"),
				Status: http.StatusOK,
			},
		},
		{
			name: "render with status code",
			args: args{
				name:    "test",
				source:  `(() => { response.render("test", 200); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Render: []byte("test"),
				Status: http.StatusOK,
			},
		},
		{
			name: "render with invalid status code",
			args: args{
				name:    "test",
				source:  `(() => { response.render("test", 9999); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Render: []byte("test"),
				Status: http.StatusInternalServerError,
			},
		},
		{
			name: "redirect without status code",
			args: args{
				name:    "test",
				source:  `(() => { response.redirect("http://test"); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Redirect:       true,
				RedirectURL:    "http://test",
				RedirectStatus: http.StatusFound,
			},
		},
		{
			name: "redirect with status code",
			args: args{
				name:    "test",
				source:  `(() => { response.redirect("http://test", 303); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Redirect:       true,
				RedirectURL:    "http://test",
				RedirectStatus: http.StatusSeeOther,
			},
		},
		{
			name: "redirect with invalid status code",
			args: args{
				name:    "test",
				source:  `(() => { response.redirect("http://test", 999); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Redirect:       true,
				RedirectURL:    "http://test",
				RedirectStatus: http.StatusInternalServerError,
			},
		},
		{
			name: "set header",
			args: args{
				name:    "test",
				source:  `(() => { response.setHeader("key", "value"); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Headers: map[string]string{
					"key": "value",
				},
			},
		},
		{
			name: "set title",
			args: args{
				name:    "test",
				source:  `(() => { response.setTitle("test"); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Title: "test",
			},
		},
		{
			name: "set meta",
			args: args{
				name:    "test",
				source:  `(() => { response.setMeta("test", {"name": "test"}); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Metas: map[string]map[string]string{
					"test": {"name": "test"},
				},
			},
		},
		{
			name: "set link",
			args: args{
				name:    "test",
				source:  `(() => { response.setLink("test", {"href": "/test"}); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Links: map[string]map[string]string{
					"test": {"href": "/test"},
				},
			},
		},
		{
			name: "set script",
			args: args{
				name:    "test",
				source:  `(() => { response.setScript("test", {"type": "test", "children": ""}); })();`,
				timeout: time.Duration(1) * time.Second,
				info:    info,
				req:     req,
				state:   nil,
			},
			want: &vmResult{
				Scripts: map[string]map[string]string{
					"test": {"type": "test", "children": ""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			err = v.Configure("test", tt.args.info, tt.args.req, tt.args.state)
			if err != nil {
				t.Errorf("failed to configure VM: %s", err)
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
