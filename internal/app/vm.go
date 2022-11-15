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
	serverRequestObject         *v8go.ObjectTemplate
	serverResponseObject        *v8go.ObjectTemplate
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
	metas          *domElementList
	links          *domElementList
	scripts        *domElementList
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
	isolate := v8go.NewIsolate()

	v := vm{
		logger:                      log.New(os.Stderr, fmt.Sprint(vmLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		isolate:                     isolate,
		processObject:               v8go.NewObjectTemplate(isolate),
		envObject:                   v8go.NewObjectTemplate(isolate),
		serverObject:                v8go.NewObjectTemplate(isolate),
		serverRequestObject:         v8go.NewObjectTemplate(isolate),
		serverResponseObject:        v8go.NewObjectTemplate(isolate),
		context:                     v8go.NewContext(isolate),
		data:                        &vmData{},
		v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
		v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
	}

	v.serverObject.Set("addr", v.v8NewFunctionTemplate(v.isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.server.Addr)
		return value
	}))

	v.serverObject.Set("port", v.v8NewFunctionTemplate(v.isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, int32(v.data.server.Port))
		return value
	}))

	v.serverObject.Set("version", v.v8NewFunctionTemplate(v.isolate,

		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.server.Version)
			return value
		}))

	v.serverRequestObject.Set("method", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.Method)
			return value
		}))

	v.serverRequestObject.Set("proto", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.Proto)
			return value
		}))

	v.serverRequestObject.Set("protoMajor", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.ProtoMajor)
			return value
		}))

	v.serverRequestObject.Set("protoMinor", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.ProtoMinor)
			return value
		}))

	v.serverRequestObject.Set("remoteAddr", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.RemoteAddr)
			return value
		}))

	v.serverRequestObject.Set("host", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.Host)
			return value
		}))

	v.serverRequestObject.Set("path", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.URL.Path)
			return value
		}))

	v.serverRequestObject.Set("query", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
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

	v.serverRequestObject.Set("headers", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
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

	v.serverRequestObject.Set("state", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.JSONParse(v.context, *v.data.state)
			return value
		}))

	v.serverResponseObject.Set("render", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 1 || !args[0].IsString() {
				return nil
			}

			r := []byte(args[0].String())
			v.data.render = &r

			status := http.StatusOK
			if len(args) > 1 && args[1].IsNumber() {
				code, err := strconv.Atoi(args[1].String())
				if err != nil || code < 100 || code > 599 {
					code = http.StatusInternalServerError
				}
				status = code
			}
			v.data.status = &status

			return nil
		}))

	v.serverResponseObject.Set("redirect", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 1 || !args[0].IsString() {
				return nil
			}

			redirect := true
			redirectURL := args[0].String()
			v.data.redirect = &redirect
			v.data.redirectURL = &redirectURL

			redirectStatus := http.StatusFound
			if len(args) > 1 && args[1].IsNumber() {
				code, err := strconv.Atoi(args[1].String())
				if err != nil || code < 100 || code > 599 {
					code = http.StatusInternalServerError
				}
				redirectStatus = code
			}
			v.data.redirectStatus = &redirectStatus

			return nil
		}))

	v.serverResponseObject.Set("setHeader", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsString() {
				return nil
			}

			key := args[0].String()
			value := args[1].String()

			if v.data.headers == nil {
				v.data.headers = make(map[string]string)
			}
			v.data.headers[key] = value

			return nil
		}))

	v.serverResponseObject.Set("setTitle", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 1 || !args[0].IsString() {
				return nil
			}

			title := args[0].String()
			v.data.title = &title

			return nil
		}))

	v.serverResponseObject.Set("setMeta", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsMap() {
				return nil
			}

			id := args[0].String()
			attributes, err := args[1].AsObject()
			if err != nil {
				return nil
			}

			e := newDOMElement(id)

			entries, err := attributes.MethodCall("entries")
			if err != nil {
				return nil
			}
			iterator, err := entries.AsObject()
			if err != nil {
				return nil
			}
			for {
				next, err := iterator.MethodCall("next")
				if err != nil {
					return nil
				}
				iteration, err := next.AsObject()
				if err != nil {
					return nil
				}
				done, err := iteration.Get("done")
				if err != nil {
					return nil
				}
				if done.Boolean() {
					break
				}
				value, err := iteration.Get("value")
				if err != nil {
					return nil
				}
				array, err := value.AsObject()
				if err != nil {
					return nil
				}
				k, err := array.GetIdx(0)
				if err != nil || !k.IsString() {
					return nil
				}
				v, err := array.GetIdx(1)
				if err != nil || !v.IsString() {
					return nil
				}
				e.SetAttribute(k.String(), v.String())
			}

			if v.data.metas == nil {
				v.data.metas = newDOMElementList()
			}
			v.data.metas.Set(e)

			return nil
		}))

	v.serverResponseObject.Set("setLink", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsMap() {
				return nil
			}

			id := args[0].String()
			attributes, err := args[1].AsObject()
			if err != nil {
				return nil
			}

			e := newDOMElement(id)

			entries, err := attributes.MethodCall("entries")
			if err != nil {
				return nil
			}
			iterator, err := entries.AsObject()
			if err != nil {
				return nil
			}
			for {
				next, err := iterator.MethodCall("next")
				if err != nil {
					return nil
				}
				iteration, err := next.AsObject()
				if err != nil {
					return nil
				}
				done, err := iteration.Get("done")
				if err != nil {
					return nil
				}
				if done.Boolean() {
					break
				}
				value, err := iteration.Get("value")
				if err != nil {
					return nil
				}
				array, err := value.AsObject()
				if err != nil {
					return nil
				}
				k, err := array.GetIdx(0)
				if err != nil || !k.IsString() {
					return nil
				}
				v, err := array.GetIdx(1)
				if err != nil || !v.IsString() {
					return nil
				}
				e.SetAttribute(k.String(), v.String())
			}

			if v.data.links == nil {
				v.data.links = newDOMElementList()
			}
			v.data.links.Set(e)

			return nil
		}))

	v.serverResponseObject.Set("setScript", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsMap() {
				return nil
			}

			id := args[0].String()
			attributes, err := args[1].AsObject()
			if err != nil {
				return nil
			}

			e := newDOMElement(id)

			entries, err := attributes.MethodCall("entries")
			if err != nil {
				return nil
			}
			iterator, err := entries.AsObject()
			if err != nil {
				return nil
			}
			for {
				next, err := iterator.MethodCall("next")
				if err != nil {
					return nil
				}
				iteration, err := next.AsObject()
				if err != nil {
					return nil
				}
				done, err := iteration.Get("done")
				if err != nil {
					return nil
				}
				if done.Boolean() {
					break
				}
				value, err := iteration.Get("value")
				if err != nil {
					return nil
				}
				array, err := value.AsObject()
				if err != nil {
					return nil
				}
				k, err := array.GetIdx(0)
				if err != nil || !k.IsString() {
					return nil
				}
				v, err := array.GetIdx(1)
				if err != nil || !v.IsString() {
					return nil
				}
				e.SetAttribute(k.String(), v.String())
			}

			if v.data.scripts == nil {
				v.data.scripts = newDOMElementList()
			}
			v.data.scripts.Set(e)

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
	serverRequest, err := v.v8ObjectTemplateNewInstance(v.serverRequestObject, v.context)
	if err != nil {
		return vmError{"failed to create template object instance"}
	}
	serverResponse, err := v.v8ObjectTemplateNewInstance(v.serverResponseObject, v.context)
	if err != nil {
		return vmError{"failed to create response object instance"}
	}

	env.Set("ENV", envName)
	process.Set("env", env)

	global := v.context.Global()
	global.Set("process", process)
	global.Set("server", server)
	global.Set("serverRequest", serverRequest)
	global.Set("serverResponse", serverResponse)

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
			if e.StackTrace != "" {
				v.logger.Printf("javascript stack trace: %+v", e)
			} else {
				v.logger.Printf("javascript error: %v", e)
			}
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
	Render         *[]byte
	Status         *int
	Redirect       *bool
	RedirectURL    *string
	RedirectStatus *int
	Headers        map[string]string
	Title          *string
	Metas          *domElementList
	Links          *domElementList
	Scripts        *domElementList
}

// newVMResult creates a new VM result
func newVMResult(d *vmData) *vmResult {
	return &vmResult{
		Render:         d.render,
		Status:         d.status,
		Redirect:       d.redirect,
		RedirectURL:    d.redirectURL,
		RedirectStatus: d.redirectStatus,
		Headers:        d.headers,
		Title:          d.title,
		Metas:          d.metas,
		Links:          d.links,
		Scripts:        d.scripts,
	}
}
