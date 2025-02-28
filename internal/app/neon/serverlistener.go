package neon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// serverListener implements a server listener.
type serverListener struct {
	name    string
	logger  *slog.Logger
	state   *serverListenerState
	server  Server
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
func newServerListener(name string, server Server) *serverListener {
	return &serverListener{
		name:   name,
		logger: slog.New(log.NewHandler(os.Stderr, "app.server.listener", nil)).With("name", name),
		state: &serverListenerState{
			sites: make(map[string]ServerSite),
		},
		server:  server,
		quit:    make(chan struct{}),
		update:  make(chan chan error),
		osClose: serverListenerOsClose,
	}
}

// Init initializes the listener.
func (l *serverListener) Init(config map[string]interface{}) error {
	l.logger.Debug("Initializing listener")

	if config == nil {
		l.logger.Error("Missing configuration")

		return errors.New("missing configuration")
	}

	var errConfig bool

	if len(config) == 0 {
		l.logger.Error("Missing listener configuration")
		errConfig = true
	}
	for listener, listenerConfig := range config {
		moduleInfo, err := module.Lookup(module.ModuleID("app.server.listener." + listener))
		if err != nil {
			l.logger.Error("Unregistered module", "module", listener)
			errConfig = true
			break
		}
		module, ok := moduleInfo.NewInstance().(core.ServerListenerModule)
		if !ok {
			l.logger.Error("Invalid module", "module", listener)
			errConfig = true
			break
		}

		moduleConfig, ok := listenerConfig.(map[string]interface{})
		if !ok {
			moduleConfig = map[string]interface{}{}
		}
		if err := module.Init(moduleConfig); err != nil {
			l.logger.Error("Failed to init module", "module", listener)
			errConfig = true
			break
		}

		l.state.listener = module

		break
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the listener.
func (l *serverListener) Register(app core.App) error {
	l.logger.Debug("Registering listener")

	mediator := newServerListenerMediator(l, app)

	if err := l.state.listener.Register(mediator); err != nil {
		return fmt.Errorf("register listener: %w", err)
	}

	l.state.mediator = mediator
	l.state.handler = newServerListenerHandler(l)

	go l.waitForEvents()

	return nil
}

// Serve starts the listener serving.
func (l *serverListener) Serve() error {
	l.logger.Debug("Accepting connections")

	if err := l.state.listener.Serve(l.state.handler); err != nil {
		return fmt.Errorf("serve listener: %w", err)
	}

	return nil
}

// Shutdown shutdowns the listener gracefully.
func (l *serverListener) Shutdown(ctx context.Context) error {
	l.logger.Debug("Shutting down listener")

	if err := l.state.listener.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown listener: %w", err)
	}

	return nil
}

// Close stops the listener listening.
func (l *serverListener) Close() error {
	l.logger.Debug("Closing listener")

	if err := l.state.listener.Close(); err != nil {
		return fmt.Errorf("close listener: %w", err)
	}

	return nil
}

// Remove removes the listener.
func (l *serverListener) Remove() error {
	l.logger.Debug("Removing listener")

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

	l.logger.Debug("Linking site", "site", site.Name())
	l.state.sites[site.Name()] = site

	errCh := make(chan error)
	l.update <- errCh
	err := <-errCh
	if err != nil {
		l.logger.Error("Failed to link site", "err", err)

		return fmt.Errorf("link site: %w", err)
	}

	return nil
}

// Unlink unlinks a site to the listener.
func (l *serverListener) Unlink(site ServerSite) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Debug("Unlinking site", "site", site.Name())
	delete(l.state.sites, site.Name())

	errCh := make(chan error)
	l.update <- errCh
	err := <-errCh
	if err != nil {
		l.logger.Error("Failed to unlink site", "err", err)

		return fmt.Errorf("unlink site: %w", err)
	}

	return nil
}

// waitForEvents waits for events.
func (l *serverListener) waitForEvents() {
	for {
		select {
		case <-l.quit:
			return

		case errCh := <-l.update:
			l.logger.Debug("New update event received, updating router")

			if err := l.updateRouter(); err != nil {
				errCh <- fmt.Errorf("update router: %w", err)
			} else {
				errCh <- nil
			}
			close(errCh)
		}
	}
}

// updateRouter updates the listener router.
func (l *serverListener) updateRouter() error {
	l.logger.Debug("Updating router")
	serverRouters := make([]ServerSiteRouter, 0, len(l.state.sites))
	for _, server := range l.state.sites {
		serverRouter, err := server.Router()
		if err != nil {
			return fmt.Errorf("get router: %w", err)
		}
		serverRouters = append(serverRouters, serverRouter)
	}

	l.state.handler.router = newServerListenerRouter(l, serverRouters...)

	return nil
}

// Listeners returns the listener listeners.
func (l *serverListener) Listeners() ([]net.Listener, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.state.mediator == nil {
		return nil, errors.New("listener not ready")
	}

	return l.state.mediator.listeners, nil
}

var _ ServerListener = (*serverListener)(nil)

// serverListenerMediator implements the server listener mediator.
type serverListenerMediator struct {
	listener  *serverListener
	app       core.App
	listeners []net.Listener
	mu        sync.RWMutex
}

// newServerListenerMediator creates a new mediator.
func newServerListenerMediator(listener *serverListener, app core.App) *serverListenerMediator {
	return &serverListenerMediator{
		listener: listener,
		app:      app,
	}
}

// Names returns the listener name.
func (m *serverListenerMediator) Name() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.listener.Name()
}

// Listeners returns the network listeners.
func (m *serverListenerMediator) Listeners() []net.Listener {
	m.mu.RLock()
	defer m.mu.RUnlock()

	listeners := m.app.Listeners()

	return listeners[m.listener.name]
}

// RegisterListener registers a listener.
func (m *serverListenerMediator) RegisterListener(listener net.Listener) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listeners = append(m.listeners, listener)

	return nil
}

var _ core.ServerListener = (*serverListenerMediator)(nil)

// serverListenerHandler implements the server listener handler.
type serverListenerHandler struct {
	logger *slog.Logger
	router ServerListenerRouter
}

// newServerListenerHandler creates a new server listener handler.
func newServerListenerHandler(l *serverListener) *serverListenerHandler {
	return &serverListenerHandler{
		logger: l.logger,
	}
}

// ServeHTTP implements the http handler.
func (h *serverListenerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.router == nil {
		h.logger.Error("No router available")

		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	h.router.ServeHTTP(w, r)
}

// serverListenerRouter implements the server listener router.
type serverListenerRouter struct {
	logger *slog.Logger
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
		logger: l.logger,
		mux:    mux,
	}
}

// ServeHTTP implements the http handler.
func (r *serverListenerRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

var _ ServerListenerRouter = (*serverListenerRouter)(nil)
