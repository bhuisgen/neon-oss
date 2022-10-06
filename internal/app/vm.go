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

// vm implements a VM
type vm struct {
	logger        *log.Logger
	isolate       *v8go.Isolate
	processObject *v8go.ObjectTemplate
	envObject     *v8go.ObjectTemplate
	serverObject  *v8go.ObjectTemplate
	context       *v8go.Context
	data          *vmData
}

// vmData implements the data of a VM
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

// NewVM creates a new VM
func NewVM() *vm {
	logger := log.New(os.Stderr, fmt.Sprint(vmLogger, ": "), log.LstdFlags|log.Lmsgprefix)
	isolate := v8go.NewIsolate()
	processObject := v8go.NewObjectTemplate(isolate)
	envObject := v8go.NewObjectTemplate(isolate)
	serverObject := v8go.NewObjectTemplate(isolate)
	context := v8go.NewContext(isolate)

	v := vm{
		logger:        logger,
		isolate:       isolate,
		processObject: processObject,
		envObject:     envObject,
		serverObject:  serverObject,
		context:       context,
		data:          &vmData{},
	}

	serverObject.Set("addr", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.server.Addr)
		return value
	}))

	serverObject.Set("port", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, int32(v.data.server.Port))
		return value
	}))

	serverObject.Set("version", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.server.Version)
		return value
	}))

	serverObject.Set("requestMethod", v8go.NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.Method)
			return value
		}))

	serverObject.Set("requestProto", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.NewValue(v.isolate, v.data.req.Proto)
		return value
	}))

	serverObject.Set("requestProtoMajor", v8go.NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.ProtoMajor)
			return value
		}))

	serverObject.Set("requestProtoMinor", v8go.NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.ProtoMinor)
			return value
		}))

	serverObject.Set("requestRemoteAddr", v8go.NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.RemoteAddr)
			return value
		}))

	serverObject.Set("requestHost", v8go.NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.Host)
			return value
		}))

	serverObject.Set("requestPath", v8go.NewFunctionTemplate(isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, _ := v8go.NewValue(v.isolate, v.data.req.URL.Path)
			return value
		}))

	serverObject.Set("requestQuery", v8go.NewFunctionTemplate(isolate,
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

	serverObject.Set("RequestHeaders", v8go.NewFunctionTemplate(isolate,
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

	serverObject.Set("state", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		value, _ := v8go.JSONParse(v.context, *v.data.state)
		return value
	}))

	serverObject.Set("render", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			render := []byte(args[0].String())
			v.data.render = &render

			if len(args) > 1 {
				status, err := strconv.Atoi(args[1].String())
				if err != nil {
					status = http.StatusOK
				}
				v.data.status = &status
			}
		}
		return nil
	}))

	serverObject.Set("redirect", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			redirect := true
			redirectURL := args[0].String()
			v.data.redirect = &redirect
			v.data.redirectURL = &redirectURL

			if len(args) > 1 {
				redirectStatus, err := strconv.Atoi(args[1].String())
				if err != nil {
					redirectStatus = http.StatusTemporaryRedirect
				}
				v.data.redirectStatus = &redirectStatus
			}
		}
		return nil
	}))

	serverObject.Set("setHeader", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
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

	serverObject.Set("setTitle", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			title := args[0].String()
			v.data.title = &title
		}
		return nil
	}))

	serverObject.Set("setMeta", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
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

	serverObject.Set("setLink", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
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

	serverObject.Set("setScript", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
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
	if info != nil {
		v.data.server = info
	} else {
		empty := ServerInfo{}
		v.data.server = &empty
	}
	if req != nil {
		v.data.req = req
	} else {
		empty := http.Request{}
		v.data.req = &empty
	}
	if state != nil {
		v.data.state = state
	} else {
		empty := "{}"
		v.data.state = &empty
	}

	global := v.context.Global()

	process, err := v.processObject.NewInstance(v.context)
	if err != nil {
		v.logger.Printf("Failed to create process instance: %s", err)

		return err
	}

	env, err := v.envObject.NewInstance(v.context)
	if err != nil {
		v.logger.Printf("Failed to create env instance: %s", err)

		return err
	}

	env.Set("ENV", envName)
	process.Set("env", env)
	global.Set("process", process)

	server, err := v.serverObject.NewInstance(v.context)
	if err != nil {
		v.logger.Printf("Failed to create server instance: %s", err)

		return err
	}

	global.Set("server", server)

	return nil
}

// Executes executes a bundle
func (v *vm) Execute(name string, source string, timeout int) (*vmResult, error) {
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
		v.logger.Printf("Failed to execute bundle: %s", err)
		if _, ok := os.LookupEnv("DEBUG"); ok {
			e := err.(*v8go.JSError)
			v.logger.Printf(e.Message)
			v.logger.Printf(e.Location)
			v.logger.Printf(e.StackTrace)
		}
		return nil, err

	case <-time.After(time.Duration(timeout) * time.Second):
		v.isolate.TerminateExecution()
		err := <-errors
		return nil, err
	}

	return newVMResult(v.data), nil
}

// vmResult implements the results of a VM
type vmResult struct {
	Render         []byte
	Status         int
	Redirect       bool
	RedirectURL    string
	RedirectStatus int
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
