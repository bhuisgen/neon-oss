// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Unauthorized copying of this file, via any medium is strictly prohibited.

package app

import (
	"errors"
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
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
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
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
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
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
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
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
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

	type fields struct {
		logger                      *log.Logger
		isolate                     *v8go.Isolate
		processObject               *v8go.ObjectTemplate
		envObject                   *v8go.ObjectTemplate
		serverObject                *v8go.ObjectTemplate
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
		context                     *v8go.Context
		data                        *vmData
		v8NewFunctionTemplate       func(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate
		v8ObjectTemplateNewInstance func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error)
	}
	type args struct {
		config *vmConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "invalid process object template",
			fields: fields{
				logger:                log.Default(),
				isolate:               isolate,
				processObject:         nil,
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          v8go.NewObjectTemplate(isolate),
				serverRequestObject:   v8go.NewObjectTemplate(isolate),
				serverResponseObject:  v8go.NewObjectTemplate(isolate),
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
				config: &vmConfig{
					Env:     "test",
					Request: nil,
					State:   nil,
				},
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
				serverRequestObject:   v8go.NewObjectTemplate(isolate),
				serverResponseObject:  v8go.NewObjectTemplate(isolate),
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
				config: &vmConfig{
					Env:     "test",
					Request: nil,
					State:   nil,
				},
			},
			wantErr: true,
		}, {
			name: "invalid server object template",
			fields: fields{
				logger:                log.Default(),
				isolate:               isolate,
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          nil,
				serverRequestObject:   v8go.NewObjectTemplate(isolate),
				serverResponseObject:  v8go.NewObjectTemplate(isolate),
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
				config: &vmConfig{
					Env:     "test",
					Request: nil,
					State:   nil,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server request object template",
			fields: fields{
				logger:                log.Default(),
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             v8go.NewObjectTemplate(isolate),
				serverRequestObject:   nil,
				serverResponseObject:  v8go.NewObjectTemplate(isolate),
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
				config: &vmConfig{
					Env:     "test",
					Request: nil,
					State:   nil,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server response object template",
			fields: fields{
				logger:                log.Default(),
				isolate:               isolate,
				processObject:         v8go.NewObjectTemplate(isolate),
				envObject:             v8go.NewObjectTemplate(isolate),
				serverObject:          v8go.NewObjectTemplate(isolate),
				serverRequestObject:   v8go.NewObjectTemplate(isolate),
				serverResponseObject:  nil,
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
				config: &vmConfig{
					Env:     "test",
					Request: nil,
					State:   nil,
				},
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
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
				context:                     tt.fields.context,
				data:                        tt.fields.data,
				v8NewFunctionTemplate:       tt.fields.v8NewFunctionTemplate,
				v8ObjectTemplateNewInstance: tt.fields.v8ObjectTemplateNewInstance,
			}
			if err := v.Configure(tt.args.config); (err != nil) != tt.wantErr {
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
		serverRequestObject         *v8go.ObjectTemplate
		serverResponseObject        *v8go.ObjectTemplate
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
				serverRequestObject:         v8go.NewObjectTemplate(isolate),
				serverResponseObject:        v8go.NewObjectTemplate(isolate),
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
				serverRequestObject:         v8go.NewObjectTemplate(isolate),
				serverResponseObject:        v8go.NewObjectTemplate(isolate),
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
				serverRequestObject:         v8go.NewObjectTemplate(isolate),
				serverResponseObject:        v8go.NewObjectTemplate(isolate),
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
				serverRequestObject:         tt.fields.serverRequestObject,
				serverResponseObject:        tt.fields.serverResponseObject,
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

func TestNewVMResult(t *testing.T) {
	render := []byte("test")
	status := http.StatusOK
	redirect := true
	redirectURL := "http://redirect"
	redirectStatus := http.StatusFound
	headers := map[string][]string{
		"key": {"value"},
	}
	title := "test"
	meta := newDOMElement("test")
	meta.SetAttribute("key", "value")
	metas := newDOMElementList()
	metas.Set(meta)
	link := newDOMElement("test")
	link.SetAttribute("key", "value")
	links := newDOMElementList()
	links.Set(link)
	script := newDOMElement("test")
	script.SetAttribute("key", "value")
	scripts := newDOMElementList()
	scripts.Set(script)

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
				Render:         &render,
				Status:         &status,
				Redirect:       &redirect,
				RedirectURL:    &redirectURL,
				RedirectStatus: &redirectStatus,
				Headers:        headers,
				Title:          &title,
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

func TestVM_APIServerHandler(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	type args struct {
		name    string
		config  vmConfig
		source  string
		timeout time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    *vmResult
		wantErr bool
	}{
		{
			name: "state",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.handler.state(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			err = v.Configure(&tt.args.config)
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

func TestVM_APIServerRequest(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	type args struct {
		name    string
		config  vmConfig
		source  string
		timeout time.Duration
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
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.method(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "proto",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.proto(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "proto major",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.protoMajor(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "proto minor",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.protoMinor(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "remote addr",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.remoteAddr(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "host",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.host(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "path",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.path(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "query",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.query(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
		{
			name: "header",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.request.headers(); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			err = v.Configure(&tt.args.config)
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

func TestVM_APIServerResponse(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	isolate := v8go.NewIsolate()
	defer isolate.Dispose()
	context := v8go.NewContext(isolate)
	defer context.Close()

	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Errorf("failed to create request: %s", err)
	}

	meta := newDOMElement("test")
	meta.SetAttribute("k1", "v1")
	meta.SetAttribute("k2", "v2")
	meta.SetAttribute("k3", "v3")
	metas := newDOMElementList()
	metas.Set(meta)

	link := newDOMElement("test")
	link.SetAttribute("k1", "v1")
	link.SetAttribute("k2", "v2")
	link.SetAttribute("k3", "v3")
	links := newDOMElementList()
	links.Set(link)

	script := newDOMElement("test")
	script.SetAttribute("k1", "v1")
	script.SetAttribute("k2", "v2")
	script.SetAttribute("k3", "v3")
	scripts := newDOMElementList()
	scripts.Set(script)

	type args struct {
		name    string
		config  vmConfig
		source  string
		timeout time.Duration
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
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.render("test"); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Render: bytePtr([]byte("test")),
				Status: intPtr(http.StatusOK),
			},
		},
		{
			name: "render with status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.render("test", 200); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Render: bytePtr([]byte("test")),
				Status: intPtr(http.StatusOK),
			},
		},
		{
			name: "render with invalid status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.render("test", 9999); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Render: bytePtr([]byte("test")),
				Status: intPtr(http.StatusInternalServerError),
			},
		},
		{
			name: "redirect without status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.redirect("http://test"); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Redirect:       boolPtr(true),
				RedirectURL:    stringPtr("http://test"),
				RedirectStatus: intPtr(http.StatusFound),
			},
		},
		{
			name: "redirect with status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.redirect("http://test", 303); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Redirect:       boolPtr(true),
				RedirectURL:    stringPtr("http://test"),
				RedirectStatus: intPtr(http.StatusSeeOther),
			},
		},
		{
			name: "redirect with invalid status code",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.redirect("http://test", 999); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Redirect:       boolPtr(true),
				RedirectURL:    stringPtr("http://test"),
				RedirectStatus: intPtr(http.StatusInternalServerError),
			},
		},
		{
			name: "set header",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setHeader("key", "value"); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Headers: map[string][]string{
					"key": {"value"},
				},
			},
		},
		{
			name: "set title",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setTitle("test"); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Title: stringPtr("test"),
			},
		},
		{
			name: "set meta",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setMeta("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Metas: metas,
			},
		},
		{
			name: "set link",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setLink("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Links: links,
			},
		},
		{
			name: "set script",
			args: args{
				name: "test",
				config: vmConfig{
					Env:     "test",
					Request: req,
					State:   stringPtr("{}"),
				},
				source:  `(() => { server.response.setScript("test", new Map([["k1", "v1"],["k2", "v2"],["k3", "v3"]])); })();`,
				timeout: time.Duration(1) * time.Second,
			},
			want: &vmResult{
				Scripts: scripts,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newVM()
			err = v.Configure(&tt.args.config)
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
