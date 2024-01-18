// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
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

// serverListener implements a server listener.
type serverListener struct {
	name    string
	config  *serverListenerConfig
	logger  *log.Logger
	state   *serverListenerState
	mu      sync.RWMutex
	quit    chan struct{}
	update  chan struct{}
	osClose func(f *os.File) error
}

// serverListenerConfig implements the server listener configuration.
type serverListenerConfig struct {
	Listener map[string]interface{}
}

// serverListenerState implements the server listener state.
type serverListenerState struct {
	sites          map[string]ServerSite
	listener       string
	listenerModule core.ServerListenerModule
	mediator       *serverListenerMediator
	handler        *serverListenerHandler
}

const (
	serverListenerLogger string = "listener"
)

// serverListenerOsClose redirects to os.Close.
func serverListenerOsClose(f *os.File) error {
	return f.Close()
}

// newServerListener creates a new server listener.
func newServerListener(name string) *serverListener {
	return &serverListener{
		name:    name,
		quit:    make(chan struct{}),
		update:  make(chan struct{}),
		osClose: serverListenerOsClose,
	}
}

// Check checks the listener configuration.
func (l *serverListener) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	for listener, listenerConfig := range config {
		moduleInfo, err := module.Lookup(module.ModuleID("server.listener." + listener))
		if err != nil {
			report = append(report, fmt.Sprintf("unregistered listener module '%s'", listener))
			continue
		}
		module, ok := moduleInfo.NewInstance().(core.ServerListenerModule)
		if !ok {
			report = append(report, fmt.Sprintf("invalid listener module '%s'", listener))
			continue
		}
		var moduleConfig map[string]interface{}
		moduleConfig, _ = listenerConfig.(map[string]interface{})
		r, err := module.Check(moduleConfig)
		if err != nil {
			for _, line := range r {
				report = append(report, fmt.Sprintf("failed to check configuration: %s", line))
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
func (l *serverListener) Load(config map[string]interface{}) error {
	l.config = &serverListenerConfig{
		Listener: config,
	}
	l.logger = log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverListenerLogger, l.name), log.LstdFlags|log.Lmsgprefix)
	l.state = &serverListenerState{
		sites: make(map[string]ServerSite),
	}

	for listener, listenerConfig := range l.config.Listener {
		moduleInfo, err := module.Lookup(module.ModuleID("server.listener." + listener))
		if err != nil {
			return err
		}
		module, ok := moduleInfo.NewInstance().(core.ServerListenerModule)
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
func (l *serverListener) Register(descriptor ServerListenerDescriptor) error {
	mediator := newServerListenerMediator(l)

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
	l.state.handler = newServerListenerHandler(l)

	go l.waitForEvents()

	return nil
}

// Serve starts the listener serving.
func (l *serverListener) Serve() error {
	err := l.state.listenerModule.Serve(l.state.handler)
	if err != nil {
		return err
	}

	return nil
}

// Shutdown shutdowns the listener gracefully.
func (l *serverListener) Shutdown(ctx context.Context) error {
	err := l.state.listenerModule.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Close stops the listener listening.
func (l *serverListener) Close() error {
	err := l.state.listenerModule.Close()
	if err != nil {
		return err
	}

	return nil
}

// Remove removes the listener.
func (l *serverListener) Remove() error {
	l.quit <- struct{}{}

	close(l.quit)
	close(l.update)

	return nil
}

// Name returns the listener name.
func (l *serverListener) Name() string {
	return l.name
}

// Link links a site to the listener.
func (l *serverListener) Link(site ServerSite) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.state.sites[site.Name()] = site

	l.update <- struct{}{}

	return nil
}

// Unlink unlinks a site to the listener.
func (l *serverListener) Unlink(site ServerSite) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.state.sites, site.Name())

	l.update <- struct{}{}

	return nil
}

// waitForEvents waits for events.
func (l *serverListener) waitForEvents() error {
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
func (l *serverListener) updateRouter() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var serverRouters []ServerSiteRouter
	for _, server := range l.state.sites {
		serverRouter, err := server.Router()
		if err != nil {
			return err
		}
		serverRouters = append(serverRouters, serverRouter)
	}

	l.state.handler.router = newServerListenerRouter(l, serverRouters...)

	return nil
}

// Descriptor returns the listener descriptor.
func (l *serverListener) Descriptor() (ServerListenerDescriptor, error) {
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
func (l *serverListener) buildDescriptor() (ServerListenerDescriptor, error) {
	descriptor := newServerListenerDescriptor()

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

var _ ServerListener = (*serverListener)(nil)

// serverListenerMediator implements the server listener mediator.
type serverListenerMediator struct {
	listener  *serverListener
	listeners []net.Listener
	mu        sync.RWMutex
}

// newServerListenerMediator creates a new server listener mediator.
func newServerListenerMediator(listener *serverListener) *serverListenerMediator {
	return &serverListenerMediator{
		listener: listener,
	}
}

// Names returns the listener name.
func (m *serverListenerMediator) Name() string {
	return m.listener.Name()
}

// Listeners returns the registered listeners.
func (m *serverListenerMediator) Listeners() []net.Listener {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.listeners
}

// RegisterListener registers a listener.
func (m *serverListenerMediator) RegisterListener(listener net.Listener) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listeners = append(m.listeners, listener)

	return nil
}

var _ core.ServerListener = (*serverListenerMediator)(nil)

// serverListenerDescriptor implements the server listener descriptor.
type serverListenerDescriptor struct {
	files []*os.File
	mu    sync.RWMutex
}

// newServerListenerDescriptor creates a new server listener descriptor.
func newServerListenerDescriptor() *serverListenerDescriptor {
	return &serverListenerDescriptor{}
}

// addFile adds a file to the descriptor.
func (d *serverListenerDescriptor) addFile(file *os.File) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.files = append(d.files, file)
}

// Files returns the descriptor files.
func (d *serverListenerDescriptor) Files() []*os.File {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.files
}

var _ ServerListenerDescriptor = (*serverListenerDescriptor)(nil)

// serverListenerRouter implements the server listener router.
type serverListenerRouter struct {
	listener *serverListener
	logger   *log.Logger
	mux      *http.ServeMux
}

// newServerListenerRouter creates a new listener router.
func newServerListenerRouter(l *serverListener, routers ...ServerSiteRouter) *serverListenerRouter {
	mux := http.NewServeMux()

	for _, router := range routers {
		for pattern, handler := range router.Routes() {
			mux.Handle(pattern, handler)
		}
	}

	return &serverListenerRouter{
		listener: l,
		logger:   log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverListenerLogger, l.name), log.LstdFlags|log.Lmsgprefix),
		mux:      mux,
	}
}

// ServeHTTP implements the http handler.
func (r *serverListenerRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// serverListenerHandler implements the server listener handler.
type serverListenerHandler struct {
	listener *serverListener
	logger   *log.Logger
	router   ServerListenerRouter
}

// newServerListenerHandler creates a new server listener handler.
func newServerListenerHandler(l *serverListener) *serverListenerHandler {
	return &serverListenerHandler{
		listener: l,
		logger:   log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverListenerLogger, l.name), log.LstdFlags|log.Lmsgprefix),
	}
}

// ServeHTTP implements the http handler.
func (h *serverListenerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.router == nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	h.router.ServeHTTP(w, r)
}

var _ ServerListenerRouter = (*serverListenerRouter)(nil)
