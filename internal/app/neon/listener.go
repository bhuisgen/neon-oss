// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// listener implements a listener.
type listener struct {
	name    string
	config  *listenerConfig
	logger  *log.Logger
	state   *listenerState
	mu      sync.RWMutex
	quit    chan struct{}
	update  chan struct{}
	osClose func(f *os.File) error
}

// listenerConfig implements the listener configuration.
type listenerConfig struct {
	Listener map[string]interface{}
}

// listenerState implements the listener state.
type listenerState struct {
	servers        map[string]Server
	listener       string
	listenerModule core.ListenerModule
	mediator       *listenerMediator
	handler        *listenerHandler
}

const (
	listenerLogger string = "listener"
)

// listenerOsClose redirects to os.Close.
func listenerOsClose(f *os.File) error {
	return f.Close()
}

// newListener creates a new listener.
func newListener(name string) *listener {
	return &listener{
		name:    name,
		quit:    make(chan struct{}),
		update:  make(chan struct{}),
		osClose: listenerOsClose,
	}
}

// Check checks the listener configuration.
func (l *listener) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	for listener, listenerConfig := range config {
		moduleInfo, err := module.Lookup(module.ModuleID("listener." + listener))
		if err != nil {
			report = append(report, fmt.Sprintf("listener '%s': unregistered listener module '%s'", l.name, listener))
			continue
		}
		module, ok := moduleInfo.NewInstance().(core.ListenerModule)
		if !ok {
			report = append(report, fmt.Sprintf("listener '%s': invalid listener module '%s'", l.name, listener))
			continue
		}
		var moduleConfig map[string]interface{}
		moduleConfig, _ = listenerConfig.(map[string]interface{})
		r, err := module.Check(moduleConfig)
		if err != nil {
			for _, line := range r {
				report = append(report, fmt.Sprintf("listener '%s': failed to check configuration: %s", l.name, line))
			}
			continue
		}

		break
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the listener.
func (l *listener) Load(config map[string]interface{}) error {
	l.config = &listenerConfig{
		Listener: config,
	}
	l.logger = log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", listenerLogger, l.name), log.LstdFlags|log.Lmsgprefix)
	l.state = &listenerState{
		servers: make(map[string]Server),
	}

	for listener, listenerConfig := range l.config.Listener {
		moduleInfo, err := module.Lookup(module.ModuleID("listener." + listener))
		if err != nil {
			return err
		}
		module, ok := moduleInfo.NewInstance().(core.ListenerModule)
		if !ok {
			return fmt.Errorf("invalid listener module '%s'", listener)
		}
		var moduleConfig map[string]interface{}
		moduleConfig, _ = listenerConfig.(map[string]interface{})
		err = module.Load(moduleConfig)
		if err != nil {
			return err
		}

		l.state.listener = listener
		l.state.listenerModule = module

		break
	}

	return nil
}

// Register registers the listener.
func (l *listener) Register(descriptor ListenerDescriptor) error {
	mediator := newListenerMediator(l)

	if descriptor != nil {
		for _, file := range descriptor.Files() {
			defer l.osClose(file)
			ln, err := net.FileListener(file)
			if err != nil {
				return err
			}
			mediator.listeners = append(mediator.listeners, ln)
		}
	}

	err := l.state.listenerModule.Register(mediator)
	if err != nil {
		return err
	}

	l.state.mediator = mediator
	l.state.handler = newListenerHandler(l)

	go l.waitForEvents()

	return nil
}

// Serve starts the listener serving.
func (l *listener) Serve() error {
	err := l.state.listenerModule.Serve(l.state.handler)
	if err != nil {
		return err
	}

	return nil
}

// Shutdown shutdown the listener gracefully.
func (l *listener) Shutdown(ctx context.Context) error {
	err := l.state.listenerModule.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Close stops the listener listening.
func (l *listener) Close() error {
	err := l.state.listenerModule.Close()
	if err != nil {
		return err
	}

	return nil
}

// Remove removes the listener.
func (l *listener) Remove() error {
	l.quit <- struct{}{}

	close(l.quit)
	close(l.update)

	return nil
}

// Name returns the listener name.
func (l *listener) Name() string {
	return l.name
}

// Link links a server to the listener.
func (l *listener) Link(server Server) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.state.servers[server.Name()] = server

	l.update <- struct{}{}

	return nil
}

// Unlink unlinks a server to the listener.
func (l *listener) Unlink(server Server) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.state.servers, server.Name())

	l.update <- struct{}{}

	return nil
}

// Descriptors returns the listener descriptors.
func (l *listener) Descriptor() (ListenerDescriptor, error) {
	if l.state.mediator == nil {
		return nil, errors.New("listener not ready")
	}

	descriptor, err := l.buildDescriptor()
	if err != nil {
		return nil, errors.New("invalid descriptor")
	}

	return descriptor, nil
}

// buildDescriptor builds the listener descriptor.
func (l *listener) buildDescriptor() (ListenerDescriptor, error) {
	descriptor := newListenerDescriptor()

	for _, listener := range l.state.mediator.listeners {
		switch v := listener.(type) {
		case *net.TCPListener:
			file, err := v.File()
			if err != nil {
				return nil, err
			}
			descriptor.addFile(file)

		case *net.UnixListener:
			file, err := v.File()
			if err != nil {
				return nil, err
			}
			descriptor.addFile(file)

		default:
			return nil, errors.New("unsupported listener")
		}
	}

	return descriptor, nil
}

// waitForEvents waits for events.
func (l *listener) waitForEvents() error {
	for {
		select {
		case <-l.quit:
			return nil

		case <-l.update:
			err := l.updateRouter()
			if err != nil {
				l.logger.Print("failed to update router")
			}
		}
	}
}

// updateRouter updates the listener router.
func (l *listener) updateRouter() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var serverRouters []ServerRouter
	for _, server := range l.state.servers {
		serverRouter, err := server.Router()
		if err != nil {
			return err
		}
		serverRouters = append(serverRouters, serverRouter)
	}

	l.state.handler.router = newListenerRouter(l, serverRouters...)

	return nil
}

var _ Listener = (*listener)(nil)

// listenerMediator implements the listener mediator.
type listenerMediator struct {
	listener  *listener
	listeners []net.Listener
	mu        sync.RWMutex
}

// newListenerMediator creates a new listener mediator.
func newListenerMediator(listener *listener) *listenerMediator {
	return &listenerMediator{
		listener: listener,
	}
}

func (m *listenerMediator) Name() string {
	return m.listener.Name()
}

// RegisterListener registers a listener.
func (m *listenerMediator) RegisterListener(listener net.Listener) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listeners = append(m.listeners, listener)

	return nil
}

// Listeners returns the registered listeners.
func (m *listenerMediator) Listeners() []net.Listener {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.listeners
}

var _ core.Listener = (*listenerMediator)(nil)

// listenerDescriptor implements the listener descriptor.
type listenerDescriptor struct {
	files []*os.File
	mu    sync.RWMutex
}

// newListenerDescriptor.
func newListenerDescriptor() *listenerDescriptor {
	return &listenerDescriptor{}
}

// addFile adds a file to the descriptor.
func (d *listenerDescriptor) addFile(file *os.File) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.files = append(d.files, file)
}

// Files returns the descriptor files.
func (d *listenerDescriptor) Files() []*os.File {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.files
}

var _ ListenerDescriptor = (*listenerDescriptor)(nil)

// listenerRouter implements the listener router.
type listenerRouter struct {
	listener *listener
	logger   *log.Logger
	mux      *http.ServeMux
}

// newListenerRouter creates a new listener router.
func newListenerRouter(l *listener, routers ...ServerRouter) *listenerRouter {
	mux := http.NewServeMux()

	for _, router := range routers {
		for pattern, handler := range router.Routes() {
			mux.Handle(pattern, handler)
		}
	}

	return &listenerRouter{
		listener: l,
		logger:   log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", listenerLogger, l.name), log.LstdFlags|log.Lmsgprefix),
		mux:      mux,
	}
}

// ServeHTTP implements the http handler.
func (r *listenerRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// listenerHandler implements the listener handler.
type listenerHandler struct {
	listener *listener
	logger   *log.Logger
	router   ListenerRouter
}

// newListenerHandler creates a new listener handler.
func newListenerHandler(l *listener) *listenerHandler {
	return &listenerHandler{
		listener: l,
		logger:   log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", listenerLogger, l.name), log.LstdFlags|log.Lmsgprefix),
	}
}

// ServeHTTP implements the http handler.
func (h *listenerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.router == nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	h.router.ServeHTTP(w, r)
}

var _ ListenerRouter = (*listenerRouter)(nil)
