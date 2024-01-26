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
	logger  *log.Logger
	state   *serverListenerState
	mu      sync.RWMutex
	quit    chan struct{}
	update  chan chan error
	osClose func(f *os.File) error
}

// serverListenerState implements the server listener state.
type serverListenerState struct {
	listener core.ServerListenerModule
	sites    map[string]ServerSite
	mediator *serverListenerMediator
	handler  *serverListenerHandler
}

// serverListenerOsClose redirects to os.Close.
func serverListenerOsClose(f *os.File) error {
	return f.Close()
}

// newServerListener creates a new server listener.
func newServerListener(name string) *serverListener {
	return &serverListener{
		name: name,
		state: &serverListenerState{
			sites: make(map[string]ServerSite),
		},
		quit:    make(chan struct{}),
		update:  make(chan chan error),
		osClose: serverListenerOsClose,
	}
}

// Init initializes the listener.
func (l *serverListener) Init(config map[string]interface{}, logger *log.Logger) error {
	l.logger = logger

	if config == nil {
		l.logger.Print("missing configuration")
		return errors.New("missing configuration")
	}

	var errInit bool

	if len(config) == 0 {
		l.logger.Print("missing listener configuration")
		errInit = true
	}
	for listener, listenerConfig := range config {
		moduleInfo, err := module.Lookup(module.ModuleID("server.listener." + listener))
		if err != nil {
			l.logger.Printf("unregistered module '%s'", listener)
			errInit = true
			break
		}
		module, ok := moduleInfo.NewInstance().(core.ServerListenerModule)
		if !ok {
			l.logger.Printf("invalid module '%s'", listener)
			errInit = true
			break
		}

		moduleConfig, ok := listenerConfig.(map[string]interface{})
		if !ok {
			moduleConfig = map[string]interface{}{}
		}
		if err := module.Init(moduleConfig, l.logger); err != nil {
			l.logger.Printf("failed to init module '%s'", listener)
			errInit = true
			break
		}

		l.state.listener = module

		break
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Register registers the listener.
func (l *serverListener) Register(descriptor ServerListenerDescriptor) error {
	mediator := newServerListenerMediator(l)

	if descriptor != nil {
		for _, file := range descriptor.Files() {
			ln, err := net.FileListener(file)
			_ = l.osClose(file)
			if err != nil {
				return err
			}
			mediator.listeners = append(mediator.listeners, ln)
		}
	}

	if err := l.state.listener.Register(mediator); err != nil {
		return err
	}

	l.state.mediator = mediator
	l.state.handler = newServerListenerHandler(l)

	go l.waitForEvents()

	return nil
}

// Serve starts the listener serving.
func (l *serverListener) Serve() error {
	if err := l.state.listener.Serve(l.state.handler); err != nil {
		return err
	}

	return nil
}

// Shutdown shutdowns the listener gracefully.
func (l *serverListener) Shutdown(ctx context.Context) error {
	if err := l.state.listener.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// Close stops the listener listening.
func (l *serverListener) Close() error {
	if err := l.state.listener.Close(); err != nil {
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
	l.state.sites[site.Name()] = site
	l.mu.Unlock()

	errChan := make(chan error)
	l.update <- errChan
	err := <-errChan
	if err != nil {
		return err
	}

	return nil
}

// Unlink unlinks a site to the listener.
func (l *serverListener) Unlink(site ServerSite) error {
	l.mu.Lock()
	delete(l.state.sites, site.Name())
	l.mu.Unlock()

	errChan := make(chan error)
	l.update <- errChan
	err := <-errChan
	if err != nil {
		return err
	}

	return nil
}

// waitForEvents waits for events.
func (l *serverListener) waitForEvents() {
	for {
		select {
		case <-l.quit:
			return

		case errChan := <-l.update:
			if err := l.updateRouter(); err != nil {
				l.logger.Printf("failed to update router: %s", err)
				errChan <- err
			} else {
				errChan <- nil
			}
			close(errChan)
		}
	}
}

// updateRouter updates the listener router.
func (l *serverListener) updateRouter() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	serverRouters := make([]ServerSiteRouter, 0, len(l.state.sites))
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
		return nil, err
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
	logger *log.Logger
	mux    *http.ServeMux
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
		logger: log.New(os.Stderr, fmt.Sprintf(l.logger.Prefix(), "router: "), log.LstdFlags|log.Lmsgprefix),
		mux:    mux,
	}
}

// ServeHTTP implements the http handler.
func (r *serverListenerRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// serverListenerHandler implements the server listener handler.
type serverListenerHandler struct {
	logger *log.Logger
	router ServerListenerRouter
}

// newServerListenerHandler creates a new server listener handler.
func newServerListenerHandler(l *serverListener) *serverListenerHandler {
	return &serverListenerHandler{
		logger: log.New(os.Stderr, fmt.Sprint(l.logger.Prefix(), "handler: "), log.LstdFlags|log.Lmsgprefix),
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
