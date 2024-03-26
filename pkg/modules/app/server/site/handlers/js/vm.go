package js

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/bhuisgen/gomonkey"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
)

// VM
type VM interface {
	Execute(config vmConfig, name string, code []byte, timeout time.Duration) (*vmResult, error)
}

// vm implements a VM.
type vm struct {
	options vmOptions
	logger  *slog.Logger
	config  *vmConfig
	data    *vmData
}

// vmOptions implements the VM options.
type vmOptions struct {
	heapMaxBytes uint
	stackSize    uint
}

// vmOptionFunc represents a vm option function.
type vmOptionFunc func(v *vm) error

// vmConfig implements the VM execution configuration.
type vmConfig struct {
	Env     string
	State   *[]byte
	Request *http.Request
	Site    core.ServerSite
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

const (
	vmLoggerID string = "app.server.site.handler.js.vm"
)

// newVM creates a new VM.
func newVM(options ...vmOptionFunc) (*vm, error) {
	v := &vm{
		logger: slog.New(log.NewHandler(os.Stderr, vmLoggerID, nil)),
		data:   &vmData{},
	}
	for _, option := range options {
		if err := option(v); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// WithHeapMaxBytes sets the maximum heap size in bytes.
func WithHeapMaxBytes(max uint) vmOptionFunc {
	return func(v *vm) error {
		v.options.heapMaxBytes = max
		return nil
	}
}

// WithStackSize sets the stack size in bytes.
func WithStackSize(size uint) vmOptionFunc {
	return func(v *vm) error {
		v.options.stackSize = size
		return nil
	}
}

// configure configures the VM.
func (v *vm) configure(context *gomonkey.Context, config *vmConfig) error {
	global, err := context.Global()
	if err != nil {
		return err
	}
	defer global.Release()

	server, err := context.DefineObject(global, "server", 0)
	if err != nil {
		return err
	}
	defer server.Release()

	if err := v.apiSite(context, server); err != nil {
		return err
	}
	if err := v.apiHandler(context, server); err != nil {
		return err
	}
	if err := v.apiRequest(context, server); err != nil {
		return err
	}
	if err := v.apiResponse(context, server); err != nil {
		return err
	}

	process, err := context.DefineObject(global, "process", 0)
	if err != nil {
		return err
	}
	defer process.Release()
	env, err := context.DefineObject(process, "env", 0)
	if err != nil {
		return err
	}
	defer env.Release()
	envName, err := gomonkey.NewValueString(context, config.Env)
	if err != nil {
		return err
	}
	defer envName.Release()
	if err := env.Set("ENV", envName); err != nil {
		return err
	}

	v.config = config
	v.data = &vmData{}

	return nil
}

// Executes executes the VM.
func (v *vm) Execute(config vmConfig, name string, code []byte, timeout time.Duration) (*vmResult, error) {
	defer v.timeTrack("Execute()", time.Now())

	ctxCh := make(chan *gomonkey.Context, 1)
	doneCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		ctx, err := gomonkey.NewContext(
			gomonkey.WithHeapMaxBytes(v.options.heapMaxBytes),
			gomonkey.WithNativeStackSize(v.options.stackSize),
		)
		if err != nil {
			errCh <- err
			return
		}
		defer ctx.Destroy()
		ctxCh <- ctx

		if err := v.configure(ctx, &config); err != nil {
			errCh <- err
			return
		}

		script, err := ctx.CompileScript(name, code)
		if err != nil {
			errCh <- err
			return
		}
		result, err := ctx.ExecuteScript(script)
		if err != nil {
			errCh <- err
			return
		}
		result.Release()
		doneCh <- struct{}{}
	}()

	ctx := <-ctxCh

	select {
	case <-doneCh:
		return newVMResult(v.data), nil

	case err := <-errCh:
		var jsError *gomonkey.JSError
		if errors.As(err, &jsError) {
			v.logger.Error("Failed to execute VM", "err", "JS error",
				"message", jsError.Message, "filename", jsError.Filename, "line", jsError.LineNumber)
		} else {
			v.logger.Error("Failed to execute VM", "err", err)
		}
		return nil, errVMExecute

	case <-time.After(timeout):
		ctx.RequestInterrupt()

		err := <-errCh
		var jsError *gomonkey.JSError
		if errors.As(err, &jsError) {
			v.logger.Error("Failed to execute VM", "err", "JS error",
				"message", jsError.Message, "filename", jsError.Filename, "line", jsError.LineNumber)
		} else {
			v.logger.Error("Failed to execute VM", "err", err)
		}
		return nil, errVMExecuteTimeout
	}
}

// timeTrack outputs the execution time of a function or code block
func (v *vm) timeTrack(label string, start time.Time) {
	elapsed := time.Since(start)
	v.logger.Debug(fmt.Sprintf("Execution of %s took %dms", label, elapsed.Milliseconds()))
}

var _ VM = (*vm)(nil)

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
	errVMBuild          = newVMError("build error")
	errVMExecute        = newVMError("execution error")
	errVMExecuteTimeout = newVMError("execution timeout")
)

var _ error = (*vmError)(nil)
