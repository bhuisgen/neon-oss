// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
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
	url            *string
	state          *string
	render         *[]byte
	status         *int
	redirect       *bool
	redirectURL    *string
	redirectStatus *int
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
	logger := log.New(os.Stdout, fmt.Sprint(vmLogger, ": "), log.LstdFlags|log.Lmsgprefix)
	isolate := v8go.NewIsolate()
	processObject := v8go.NewObjectTemplate(isolate)
	envObject := v8go.NewObjectTemplate(isolate)
	serverObject := v8go.NewObjectTemplate(isolate)
	context := v8go.NewContext(isolate)

	vm := vm{
		logger:        logger,
		isolate:       isolate,
		processObject: processObject,
		envObject:     envObject,
		serverObject:  serverObject,
		context:       context,
		data:          &vmData{},
	}

	serverObject.Set("url", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v, _ := v8go.NewValue(vm.isolate, *vm.data.url)

		return v
	}))

	serverObject.Set("state", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v, _ := v8go.JSONParse(vm.context, *vm.data.state)

		return v
	}))

	serverObject.Set("render", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			render := []byte(args[0].String())
			vm.data.render = &render

			if len(args) > 1 {
				status, err := strconv.Atoi(args[1].String())
				if err != nil {
					status = http.StatusOK
				}
				vm.data.status = &status
			}
		}
		return nil
	}))

	serverObject.Set("redirect", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			redirect := true
			redirectURL := args[0].String()
			vm.data.redirect = &redirect
			vm.data.redirectURL = &redirectURL

			if len(args) > 1 {
				redirectStatus, err := strconv.Atoi(args[1].String())
				if err != nil {
					redirectStatus = http.StatusTemporaryRedirect
				}
				vm.data.redirectStatus = &redirectStatus
			}
		}
		return nil
	}))

	serverObject.Set("setTitle", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			title := args[0].String()
			vm.data.title = &title
		}
		return nil
	}))

	serverObject.Set("setMeta", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 1 {
			id := args[0].String()
			data := args[1].Object()

			if vm.data.metas == nil {
				vm.data.metas = make(map[string]map[string]string)
			}

			vm.data.metas[id] = make(map[string]string)
			for _, attribute := range []string{"name", "itemprop", "content", "charset", "http-equiv", "scheme", "property"} {
				if ok := data.Has(attribute); ok {
					v, _ := data.Get(attribute)
					vm.data.metas[id][attribute] = v.String()
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

			if vm.data.links == nil {
				vm.data.links = make(map[string]map[string]string)
			}

			vm.data.links[id] = make(map[string]string)
			for _, attribute := range []string{"rel", "href", "hreflang", "type", "sizes", "media", "as",
				"crossorigin", "disabled", "importance", "integrity", "referrerpolicy", "title"} {
				if ok := data.Has(attribute); ok {
					v, _ := data.Get(attribute)
					vm.data.links[id][attribute] = v.String()
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

			if vm.data.scripts == nil {
				vm.data.scripts = make(map[string]map[string]string)
			}

			vm.data.scripts[id] = make(map[string]string)
			for _, attribute := range []string{"type", "src", "async", "crossorigin", "defer", "integrity", "nomodule",
				"nonce", "referrerpolicy", "children"} {
				if ok := data.Has(attribute); ok {
					v, _ := data.Get(attribute)
					vm.data.scripts[id][attribute] = v.String()
				}
			}
		}
		return nil
	}))

	return &vm
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
func (v *vm) Configure(envName string, req *http.Request, state *string) error {
	v.data.url = &req.URL.Path
	v.data.state = state

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
