package js

import (
	"errors"
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
	status                      vmStatus
	config                      *vmConfig
	logger                      *slog.Logger
	data                        *vmData
	v8NewFunctionTemplate       func(isolate *v8go.Isolate, callback v8go.FunctionCallback) *v8go.FunctionTemplate
	v8ObjectTemplateNewInstance func(template *v8go.ObjectTemplate, context *v8go.Context) (*v8go.Object, error)
}

// vmStatus implements the VM status.
type vmStatus int

const (
	vmStatusNew = iota
	vmStatusConfigured
)

// vmConfig implements the VM execution configuration.
type vmConfig struct {
	Env     string
	Request *http.Request
	State   *string
}

// vmData implements the VM execution data.
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

// Reset resets the VM.
func (v *vm) Reset() {
	v.config = nil
	v.logger = nil
	v.data = nil
}

// Configure configures the VM
func (v *vm) Configure(config *vmConfig, logger *slog.Logger) error {
	if v.status == vmStatusNew {
		if err := api(v); err != nil {
			return errVMConfigure
		}
	}

	process, err := v.v8ObjectTemplateNewInstance(v.processObject, v.context)
	if err != nil {
		return errVMConfigure
	}
	env, err := v.v8ObjectTemplateNewInstance(v.envObject, v.context)
	if err != nil {
		return errVMConfigure
	}
	server, err := v.v8ObjectTemplateNewInstance(v.serverObject, v.context)
	if err != nil {
		return errVMConfigure
	}
	serverHandler, err := v.v8ObjectTemplateNewInstance(v.serverHandlerObject, v.context)
	if err != nil {
		return errVMConfigure
	}
	serverRequest, err := v.v8ObjectTemplateNewInstance(v.serverRequestObject, v.context)
	if err != nil {
		return errVMConfigure
	}
	serverResponse, err := v.v8ObjectTemplateNewInstance(v.serverResponseObject, v.context)
	if err != nil {
		return errVMConfigure
	}

	if err := env.Set("ENV", config.Env); err != nil {
		return errVMConfigure
	}
	if err := process.Set("env", env); err != nil {
		return errVMConfigure
	}
	if err := server.Set("handler", serverHandler); err != nil {
		return errVMConfigure
	}
	if err := server.Set("request", serverRequest); err != nil {
		return errVMConfigure
	}
	if err := server.Set("response", serverResponse); err != nil {
		return errVMConfigure
	}

	global := v.context.Global()
	if err := global.Set("process", process); err != nil {
		return errVMConfigure
	}
	if err := global.Set("server", server); err != nil {
		return errVMConfigure
	}

	v.status = vmStatusConfigured

	v.config = config
	v.logger = logger
	v.data = &vmData{}

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
	vals := make(chan *v8go.Value, 1)
	errs := make(chan error, 1)

	go worker(vals, errs)
	select {
	case <-vals:

	case err := <-errs:
		var jsError *v8go.JSError
		if errors.As(err, &jsError) {
			v.logger.Debug("Failed to execute script", "name", name, "message", jsError.Message,
				"location", jsError.Location, "stackTrace", jsError.StackTrace)
		} else {
			v.logger.Debug("Failed to execute script", "name", name, "err", err)
		}
		return nil, errVMExecute

	case <-time.After(timeout):
		v.isolate.TerminateExecution()
		<-errs
		return nil, errVMExecutionTimeout
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
	errVMConfigure        = newVMError("configuration error")
	errVMExecute          = newVMError("execution error")
	errVMExecutionTimeout = newVMError("execution timeout")
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
