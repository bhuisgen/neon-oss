// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"rogchap.com/v8go"
)

// VM
type VM interface {
	Close()
	Reset()
	Configure(envName string, info *ServerInfo, req *http.Request, state *string) error
	Execute(name string, source string, timeout time.Duration) (*vmResult, error)
}

// vm implements a VM
type vm struct {
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

// vmData implements the VM data
type vmData struct {
	server         *ServerInfo
	req            *http.Request
	state          *string
	render         *[]byte
	status         *int
	redirect       *bool
	redirectURL    *string
	redirectStatus *int
	headers        map[string]string
	title          *string
	metas          map[string]map[string]string
	links          map[string]map[string]string
	scripts        map[string]map[string]string
}

const (
	vmLogger string = "vm"
)

// vmV8NewFunctionTemplate redirects to v8go.NewFunctionTemplate
func vmV8NewFunctionTemplate(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(isolate, callback)
}

// vmV8ObjectTemplateNewInstance redirects to v8go.ObjectTemplate.NewInstance
func vmV8ObjectTemplateNewInstance(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error) {
	return template.NewInstance(context)
}

// newVM creates a new VM
func newVM() *vm {
	logger := log.New(os.Stderr, fmt.Sprint(vmLogger, ": "), log.LstdFlags|log.Lmsgprefix)
	isolate := v8go.NewIsolate()
	processObject := v8go.NewObjectTemplate(isolate)
	envObject := v8go.NewObjectTemplate(isolate)
	serverObject := v8go.NewObjectTemplate(isolate)
	requestObject := v8go.NewObjectTemplate(isolate)
	responseObject := v8go.NewObjectTemplate(isolate)
	context := v8go.NewContext(isolate)

	v := vm{
		logger:                      logger,
		isolate:                     isolate,
		processObject:               processObject,
		envObject:                   envObject,
		serverObject:                serverObject,
		requestObject:               requestObject,
		responseObject:              responseObject,
		context:                     context,
		data:                        &vmData{},
		v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
		v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
	}

	serverObject.Set("addr", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.server.Addr)
		return value
	}))

	serverObject.Set("port", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, int32(v.data.server.Port))
		return value
	}))

	serverObject.Set("version", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.server.Version)
		return value
	}))

	requestObject.Set("method", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.Method)
		return value
	}))

	requestObject.Set("proto", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.Proto)
		return value
	}))

	requestObject.Set("protoMajor", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.ProtoMajor)
		return value
	}))

	requestObject.Set("protoMinor", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.ProtoMinor)
		return value
	}))

	requestObject.Set("remoteAddr", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.RemoteAddr)
		return value
	}))

	requestObject.Set("host", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.Host)
		return value
	}))

	requestObject.Set("path", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.URL.Path)
		return value
	}))

	requestObject.Set("query", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		var value *v8go.Value
		q := v.data.req.URL.Query()
		data, err := json.Marshal(&q)
		if err != nil {
			value, _ = v8go.JSONParse(v.context, "{}")
		} else {
			value, _ = v8go.JSONParse(v.context, string(data))
		}
		return value
	}))

	requestObject.Set("headers", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		var value *v8go.Value
		h := v.data.req.Header
		data, err := json.Marshal(&h)
		if err != nil {
			value, _ = v8go.JSONParse(v.context, "{}")
		} else {
			value, _ = v8go.JSONParse(v.context, string(data))
		}
		return value
	}))

	requestObject.Set("state", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.JSONParse(v.context, *v.data.state)
		return value
	}))

	responseObject.Set("render", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			render := []byte(args[0].String())
			v.data.render = &render

			status := http.StatusOK
			if len(args) > 1 {
				code, err := strconv.Atoi(args[1].String())
				if err != nil || code < 100 || code > 599 {
					code = http.StatusInternalServerError
				}
				status = code
			}
			v.data.status = &status
		}
		return nil
	}))

	responseObject.Set("redirect", v.v8NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 0 {
				redirect := true
				redirectURL := args[0].String()
				v.data.redirect = &redirect
				v.data.redirectURL = &redirectURL

				redirectStatus := http.StatusFound
				if len(args) > 1 {
					code, err := strconv.Atoi(args[1].String())
					if err != nil || code < 100 || code > 599 {
						code = http.StatusInternalServerError
					}
					redirectStatus = code
				}
				v.data.redirectStatus = &redirectStatus
			}
			return nil
		}))

	responseObject.Set("setHeader", v.v8NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 1 {
				key := args[0].String()
				value := args[1].String()

				if v.data.headers == nil {
					v.data.headers = make(map[string]string)
				}

				v.data.headers[key] = value
			}

			return nil
		}))

	responseObject.Set("setTitle", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			title := args[0].String()
			v.data.title = &title
		}
		return nil
	}))

	responseObject.Set("setMeta", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 1 {
			id := args[0].String()
			data := args[1].Object()

			if v.data.metas == nil {
				v.data.metas = make(map[string]map[string]string)
			}

			v.data.metas[id] = make(map[string]string)
			for _, attribute := range []string{"name", "itemprop", "content", "charset", "http-equiv", "scheme", "property"} {
				if ok := data.Has(attribute); ok {
					value, _ := data.Get(attribute)
					v.data.metas[id][attribute] = value.String()
				}
			}
		}
		return nil
	}))

	responseObject.Set("setLink", v.v8NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 1 {
			id := args[0].String()
			data := args[1].Object()

			if v.data.links == nil {
				v.data.links = make(map[string]map[string]string)
			}

			v.data.links[id] = make(map[string]string)
			for _, attribute := range []string{"rel", "href", "hreflang", "type", "sizes", "media", "as",
				"crossorigin", "disabled", "importance", "integrity", "referrerpolicy", "title"} {
				if ok := data.Has(attribute); ok {
					value, _ := data.Get(attribute)
					v.data.links[id][attribute] = value.String()
				}
			}
		}
		return nil
	}))

	responseObject.Set("setScript", v.v8NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 1 {
				id := args[0].String()
				data := args[1].Object()

				if v.data.scripts == nil {
					v.data.scripts = make(map[string]map[string]string)
				}

				v.data.scripts[id] = make(map[string]string)
				for _, attribute := range []string{"type", "src", "async", "crossorigin", "defer", "integrity", "nomodule",
					"nonce", "referrerpolicy", "children"} {
					if ok := data.Has(attribute); ok {
						value, _ := data.Get(attribute)
						v.data.scripts[id][attribute] = value.String()
					}
				}
			}
			return nil
		}))

	return &v
}

// Close closes the VM
func (v *vm) Close() {
	v.context.Close()
	v.isolate.Dispose()
}

// Reset resets the VM cache and data
func (v *vm) Reset() {
	v.data = &vmData{}
}

// Configure configures the VM
func (v *vm) Configure(envName string, info *ServerInfo, req *http.Request, state *string) error {
	if info == nil {
		return vmError{"invalid request"}
	}
	v.data.server = info

	if req == nil {
		return vmError{"invalid server informations"}
	}
	v.data.req = req

	if state != nil {
		v.data.state = state
	} else {
		empty := "{}"
		v.data.state = &empty
	}

	process, err := v.v8ObjectTemplateNewInstance(v.processObject, v.context)
	if err != nil {
		return vmError{"failed to create process object instance"}
	}
	env, err := v.v8ObjectTemplateNewInstance(v.envObject, v.context)
	if err != nil {
		return vmError{"failed to create env object instance"}
	}
	server, err := v.v8ObjectTemplateNewInstance(v.serverObject, v.context)
	if err != nil {
		return vmError{"failed to create server object instance"}
	}
	request, err := v.v8ObjectTemplateNewInstance(v.requestObject, v.context)
	if err != nil {
		return vmError{"failed to create template object instance"}
	}
	response, err := v.v8ObjectTemplateNewInstance(v.responseObject, v.context)
	if err != nil {
		return vmError{"failed to create response object instance"}
	}

	env.Set("ENV", envName)
	process.Set("env", env)

	global := v.context.Global()
	global.Set("process", process)
	global.Set("server", server)
	global.Set("request", request)
	global.Set("response", response)

	return nil
}

// Executes executes a bundle
func (v *vm) Execute(name string, source string, timeout time.Duration) (*vmResult, error) {
	worker := func(values chan<- *v8go.Value, errors chan<- error) {
		value, err := v.context.RunScript(source, name)
		if err != nil {
			errors <- err
			return
		}
		values <- value
	}
	values := make(chan *v8go.Value, 1)
	errors := make(chan error, 1)

	go worker(values, errors)
	select {
	case <-values:

	case err := <-errors:
		if DEBUG {
			e := err.(*v8go.JSError)
			v.logger.Printf(e.Message)
			v.logger.Printf(e.Location)
			v.logger.Printf(e.StackTrace)
		}
		return nil, vmError{"execution error"}

	case <-time.After(timeout):
		v.isolate.TerminateExecution()
		<-errors
		return nil, vmError{"execution timeout"}
	}

	return newVMResult(v.data), nil
}

// vmError implements a VM error
type vmError struct {
	message string
}

// Error returns the error message
func (e vmError) Error() string {
	return e.message
}

// vmResult implements the results of a VM
type vmResult struct {
	Render         []byte
	Status         int
	Redirect       bool
	RedirectURL    string
	RedirectStatus int
	Headers        map[string]string
	Title          string
	Metas          map[string]map[string]string
	Links          map[string]map[string]string
	Scripts        map[string]map[string]string
}

// newVMResult creates a new VM result
func newVMResult(d *vmData) *vmResult {
	r := vmResult{}
	if d.render != nil {
		r.Render = *d.render
	}
	if d.status != nil {
		r.Status = *d.status
	}
	if d.redirect != nil {
		r.Redirect = *d.redirect
	}
	if d.redirectURL != nil {
		r.RedirectURL = *d.redirectURL
	}
	if d.redirectStatus != nil {
		r.RedirectStatus = *d.redirectStatus
	}
	if d.headers != nil {
		r.Headers = d.headers
	}
	if d.title != nil {
		r.Title = *d.title
	}
	if d.metas != nil {
		r.Metas = copyMap(d.metas)
	}
	if d.links != nil {
		r.Links = copyMap(d.links)
	}
	if d.scripts != nil {
		r.Scripts = copyMap(d.scripts)
	}
	return &r
}

// copyMap copy a map into a new one
func copyMap(m map[string]map[string]string) map[string]map[string]string {
	var cp map[string]map[string]string
	if m != nil {
		cp = make(map[string]map[string]string)
		for k1, v1 := range m {
			cp[k1] = make(map[string]string)
			for k2, v2 := range v1 {
				cp[k1][k2] = v2
			}
		}
	}
	return cp
}
