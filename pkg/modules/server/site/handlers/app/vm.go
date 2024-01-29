// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"rogchap.com/v8go"
)

// VM
type VM interface {
	Close()
	Reset()
	Configure(config *vmConfig, logger *slog.Logger) error
	Execute(name string, source string, timeout time.Duration) (*vmResult, error)
}

// vm implements a VM.
type vm struct {
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

// vmConfig implements the VM configuration.
type vmConfig struct {
	Env     string
	Request *http.Request
	State   *string
}

// vmStatus implements the VM status.
type vmStatus int

const (
	vmStatusNew = iota
	vmStatusReady
)

// vmData implements the VM data.
type vmData struct {
	render         *[]byte
	status         *int
	redirect       *bool
	redirectURL    *string
	redirectStatus *int
	headers        map[string][]string
	title          *string
	metas          *domElementList
	links          *domElementList
	scripts        *domElementList
}

const (
	vmLogger string = "vm"
)

// vmV8NewFunctionTemplate redirects to v8go.NewFunctionTemplate.
func vmV8NewFunctionTemplate(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(isolate, callback)
}

// vmV8ObjectTemplateNewInstance redirects to v8go.ObjectTemplate.NewInstance.
func vmV8ObjectTemplateNewInstance(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error) {
	return template.NewInstance(context)
}

// newVM creates a new VM.
func newVM() *vm {
	isolate := v8go.NewIsolate()
	return &vm{
		isolate:                     isolate,
		processObject:               v8go.NewObjectTemplate(isolate),
		envObject:                   v8go.NewObjectTemplate(isolate),
		serverObject:                v8go.NewObjectTemplate(isolate),
		serverHandlerObject:         v8go.NewObjectTemplate(isolate),
		serverRequestObject:         v8go.NewObjectTemplate(isolate),
		serverResponseObject:        v8go.NewObjectTemplate(isolate),
		context:                     v8go.NewContext(isolate),
		status:                      vmStatusNew,
		data:                        &vmData{},
		v8NewFunctionTemplate:       vmV8NewFunctionTemplate,
		v8ObjectTemplateNewInstance: vmV8ObjectTemplateNewInstance,
	}
}

// Close closes the VM.
func (v *vm) Close() {
	v.context.Close()
	v.isolate.Dispose()
}

// Reset resets the VM cache and data.
func (v *vm) Reset() {
	v.data = &vmData{}
}

// Configure configures the VM
func (v *vm) Configure(config *vmConfig, logger *slog.Logger) error {
	if v.status == vmStatusNew {
		if err := api(v); err != nil {
			return vmErrConfigure
		}
		v.status = vmStatusReady
	}

	process, err := v.v8ObjectTemplateNewInstance(v.processObject, v.context)
	if err != nil {
		return vmErrConfigure
	}
	env, err := v.v8ObjectTemplateNewInstance(v.envObject, v.context)
	if err != nil {
		return vmErrConfigure
	}
	server, err := v.v8ObjectTemplateNewInstance(v.serverObject, v.context)
	if err != nil {
		return vmErrConfigure
	}
	serverHandler, err := v.v8ObjectTemplateNewInstance(v.serverHandlerObject, v.context)
	if err != nil {
		return vmErrConfigure
	}
	serverRequest, err := v.v8ObjectTemplateNewInstance(v.serverRequestObject, v.context)
	if err != nil {
		return vmErrConfigure
	}
	serverResponse, err := v.v8ObjectTemplateNewInstance(v.serverResponseObject, v.context)
	if err != nil {
		return vmErrConfigure
	}

	if err := env.Set("ENV", config.Env); err != nil {
		return vmErrConfigure
	}
	if err := process.Set("env", env); err != nil {
		return vmErrConfigure
	}
	if err := server.Set("handler", serverHandler); err != nil {
		return vmErrConfigure
	}
	if err := server.Set("request", serverRequest); err != nil {
		return vmErrConfigure
	}
	if err := server.Set("response", serverResponse); err != nil {
		return vmErrConfigure
	}

	global := v.context.Global()
	if err := global.Set("process", process); err != nil {
		return vmErrConfigure
	}
	if err := global.Set("server", server); err != nil {
		return vmErrConfigure
	}

	v.config = config
	v.logger = logger

	return nil
}

// Executes executes a script.
func (v *vm) Execute(name string, source string, timeout time.Duration) (*vmResult, error) {
	defer v.timeTrack("Execute()", time.Now())

	worker := func(vals chan<- *v8go.Value, errs chan<- error) {
		value, err := v.context.RunScript(source, name)
		if err != nil {
			errs <- err
			return
		}
		vals <- value
	}
	jsVal := make(chan *v8go.Value, 1)
	jsErr := make(chan error, 1)

	go worker(jsVal, jsErr)
	select {
	case <-jsVal:

	case err := <-jsErr:
		var jsError *v8go.JSError
		if errors.As(err, &jsError) {
			v.logger.Debug("Failed to execute script", "name", name, "stackTrace", fmt.Sprintf("%+v", jsError))
		}
		return nil, vmErrExecute

	case <-time.After(timeout):
		v.isolate.TerminateExecution()
		<-jsErr
		return nil, vmErrExecutionTimeout
	}

	return newVMResult(v.data), nil
}

// timeTrack outputs the execution time of a function or code block
func (v *vm) timeTrack(label string, start time.Time) {
	elapsed := time.Since(start)
	v.logger.Debug("Execution of %s took %s", label, elapsed)
}

var _ VM = (*vm)(nil)

// vmError implements a VM error.
type vmError struct {
	message string
}

// newVMError creates a new error.
func newVMError(message string) *vmError {
	return &vmError{
		message: message,
	}
}

// Error returns the error message.
func (e vmError) Error() string {
	return e.message
}

var (
	vmErrConfigure        = newVMError("configuration error")
	vmErrExecute          = newVMError("execution error")
	vmErrExecutionTimeout = newVMError("execution timeout")
)

var _ error = (*vmError)(nil)

// vmResult implements the results of a VM.
type vmResult struct {
	Render         *[]byte
	Status         *int
	Redirect       *bool
	RedirectURL    *string
	RedirectStatus *int
	Headers        map[string][]string
	Title          *string
	Metas          *domElementList
	Links          *domElementList
	Scripts        *domElementList
}

// newVMResult creates a new VM result.
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
