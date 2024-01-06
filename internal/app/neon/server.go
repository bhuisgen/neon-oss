// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// server implements a server.
type server struct {
	name    string
	config  *serverConfig
	logger  *log.Logger
	state   *serverState
	store   Store
	fetcher Fetcher
	mu      sync.RWMutex
}

// serverConfig implements the server configuration.
type serverConfig struct {
	Listeners []string
	Hosts     []string
	Routes    map[string]serverRouteConfig
}

// serverRouteConfig implements a server route configuration.
type serverRouteConfig struct {
	Middlewares map[string]interface{}
	Handler     map[string]interface{}
}

// serverState implements the server state.
type serverState struct {
	listeners     []string
	hosts         []string
	defaultServer bool
	routes        []string
	routesMap     map[string]serverRouteState
	mediator      *serverMediator
	router        *serverRouter
	middleware    *serverMiddleware
	handler       *serverHandler
}

// serverRouteState implements a server route state.
type serverRouteState struct {
	middlewares        []string
	middlewaresModules map[string]core.ServerMiddlewareModule
	handler            string
	handlerModule      core.ServerHandlerModule
}

const (
	serverLogger       string = "server"
	serverRouteDefault string = "default"
)

// newServer creates a new server.
func newServer(name string, store Store, fetcher Fetcher) *server {
	return &server{
		name:    name,
		store:   store,
		fetcher: fetcher,
	}
}

// Check checks the server configuration.
func (s *server) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c serverConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "server: failed to parse configuration")
		return report, err
	}

	if len(c.Listeners) == 0 {
		report = append(report, "server: no listener defined")
	}

	for _, routeConfig := range c.Routes {
		for middleware, middlewareConfig := range routeConfig.Middlewares {
			moduleInfo, err := module.Lookup(module.ModuleID("server.middleware." + middleware))
			if err != nil {
				report = append(report, fmt.Sprintf("server: unregistered middleware module '%s'", middleware))
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.ServerMiddlewareModule)
			if !ok {
				report = append(report, fmt.Sprintf("server: invalid middleware module '%s'", middleware))
				continue
			}
			var moduleConfig map[string]interface{}
			moduleConfig, _ = middlewareConfig.(map[string]interface{})
			r, err := module.Check(moduleConfig)
			if err != nil {
				for _, line := range r {
					report = append(report, fmt.Sprintf("server: middleware '%s', failed to check configuration: %s", middleware,
						line))
					continue
				}
			}
		}

		for handler, handlerConfig := range routeConfig.Handler {
			moduleInfo, err := module.Lookup(module.ModuleID("server.handler." + handler))
			if err != nil {
				report = append(report, fmt.Sprintf("server: unregistered handler module '%s'", handler))
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.ServerHandlerModule)
			if !ok {
				report = append(report, fmt.Sprintf("server: invalid handler module '%s'", handler))
				continue
			}
			var moduleConfig map[string]interface{}
			moduleConfig, _ = handlerConfig.(map[string]interface{})
			r, err := module.Check(moduleConfig)
			if err != nil {
				for _, line := range r {
					report = append(report, fmt.Sprintf("server: handler '%s', failed to check configuration: %s", handler, line))
				}
				continue
			}
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the server.
func (s *server) Load(config map[string]interface{}) error {
	var c serverConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	s.config = &c
	s.logger = log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverLogger, s.name), log.LstdFlags|log.Lmsgprefix)
	s.state = &serverState{
		routesMap: make(map[string]serverRouteState),
	}

	s.state.hosts = append(s.state.hosts, s.config.Hosts...)
	s.state.listeners = append(s.state.listeners, s.config.Listeners...)

	for route, routeConfig := range s.config.Routes {
		stateRoute := serverRouteState{
			middlewaresModules: make(map[string]core.ServerMiddlewareModule),
		}

		for middleware, middlewareConfig := range routeConfig.Middlewares {
			moduleInfo, err := module.Lookup(module.ModuleID("server.middleware." + middleware))
			if err != nil {
				return err
			}
			module, ok := moduleInfo.NewInstance().(core.ServerMiddlewareModule)
			if !ok {
				return fmt.Errorf("invalid middleware module '%s'", middleware)
			}
			var moduleConfig map[string]interface{}
			moduleConfig, _ = middlewareConfig.(map[string]interface{})
			err = module.Load(moduleConfig)
			if err != nil {
				return err
			}

			stateRoute.middlewares = append(stateRoute.middlewares, middleware)
			stateRoute.middlewaresModules[middleware] = module
		}

		for handler, handlerConfig := range routeConfig.Handler {
			moduleInfo, err := module.Lookup(module.ModuleID("server.handler." + handler))
			if err != nil {
				return err
			}
			module, ok := moduleInfo.NewInstance().(core.ServerHandlerModule)
			if !ok {
				return fmt.Errorf("invalid handler module '%s'", handler)
			}
			var moduleConfig map[string]interface{}
			moduleConfig, _ = handlerConfig.(map[string]interface{})
			err = module.Load(moduleConfig)
			if err != nil {
				return err
			}

			stateRoute.handler = handler
			stateRoute.handlerModule = module
		}

		s.state.routes = append(s.state.routes, route)
		s.state.routesMap[route] = stateRoute
	}

	return nil
}

// Register registers the middlewares and handlers.
func (s *server) Register() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mediator := newServerMediator(s)

	for _, route := range s.state.routes {
		mediator.currentRoute = route

		for _, middleware := range s.state.routesMap[route].middlewares {
			err := s.state.routesMap[route].middlewaresModules[middleware].Register(mediator)
			if err != nil {
				return err
			}
		}

		err := s.state.routesMap[route].handlerModule.Register(mediator)
		if err != nil {
			return err
		}
	}

	s.state.mediator = mediator

	router, err := s.buildRouter()
	if err != nil {
		return err
	}

	s.state.router = router
	s.state.middleware = newServerMiddleware(s)
	s.state.handler = newServerHandler(s)

	return nil
}

// Start starts the server.
func (s *server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, route := range s.state.routes {
		for _, middleware := range s.state.routesMap[route].middlewares {
			err := s.state.routesMap[route].middlewaresModules[middleware].Start()
			if err != nil {
				return err
			}
		}

		err := s.state.routesMap[route].handlerModule.Start()
		if err != nil {
			return err
		}
	}

	return nil
}

// Enable enables the server.
func (s *server) Enable() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, route := range s.state.routes {
		for _, middleware := range s.state.routesMap[route].middlewares {
			err := s.state.routesMap[route].middlewaresModules[middleware].Mount()
			if err != nil {
				return err
			}
		}

		err := s.state.routesMap[route].handlerModule.Mount()
		if err != nil {
			return err
		}
	}

	return nil
}

// Disable disables the server.
func (s *server) Disable(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, route := range s.state.routes {
		for _, middleware := range s.state.routesMap[route].middlewares {
			s.state.routesMap[route].middlewaresModules[middleware].Unmount()
		}

		s.state.routesMap[route].handlerModule.Unmount()
	}

	return nil
}

// Stop stops the server.
func (s *server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, route := range s.state.routes {
		for _, middleware := range s.state.routesMap[route].middlewares {
			s.state.routesMap[route].middlewaresModules[middleware].Stop()
		}

		s.state.routesMap[route].handlerModule.Stop()
	}

	return nil
}

// Remove removes the server.
func (s *server) Remove() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return nil
}

// Name returns the server name.
func (s *server) Name() string {
	return s.name
}

// Listeners returns the server listeners.
func (s *server) Listeners() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.listeners
}

// Hosts returns the server hosts.
func (s *server) Hosts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.hosts
}

// Default returns true if the server is the default server.
func (s *server) Default() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.defaultServer
}

// Router returns the server router.
func (s *server) Router() (ServerRouter, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state.router == nil {
		return nil, errors.New("server not ready")
	}

	return s.state.router, nil
}

// buildRouter builds the server router.
func (s *server) buildRouter() (*serverRouter, error) {
	routes := make(map[string]http.Handler, len(s.state.routes))

	var findRouteMiddlewares func(route string) []func(http.Handler) http.Handler
	findRouteMiddlewares = func(route string) []func(http.Handler) http.Handler {
		if middlewares, ok := s.state.mediator.routesMiddlewares[route]; ok {
			return middlewares
		}
		if route == "/" {
			return s.state.mediator.defaultMiddlewares
		}
		return findRouteMiddlewares(path.Dir(route))
	}

	for _, route := range s.state.routes {
		if !strings.HasPrefix(route, "/") {
			continue
		}

		var handler http.Handler
		if h, ok := (s.state.mediator.routesHandler[route]); ok {
			handler = h
		} else {
			handler = s.state.handler
		}
		routeMiddlewares := findRouteMiddlewares(route)
		for i := len(routeMiddlewares) - 1; i >= 0; i-- {
			handler = routeMiddlewares[i](handler)
		}
		routes[route] = s.state.middleware.Handler(handler)
	}

	if _, ok := s.state.mediator.routesHandler["/"]; !ok {
		var handler http.Handler

		if s.state.mediator.defaultHandler != nil {
			handler = s.state.mediator.defaultHandler
		} else {
			handler = s.state.handler
		}
		routeMiddlewares := findRouteMiddlewares("/")
		for i := len(routeMiddlewares) - 1; i >= 0; i-- {
			handler = routeMiddlewares[i](handler)
		}
		routes["/"] = s.state.middleware.Handler(handler)
	}

	router := newServerRouter()

	if len(s.state.hosts) > 0 {
		for _, name := range s.state.hosts {
			for route, handler := range routes {
				router.addRoute(name+route, handler)
			}
		}
	} else {
		for route, handler := range routes {
			router.addRoute(route, handler)
		}
	}

	return router, nil
}

var _ Server = (*server)(nil)

// serverMediator implements the server mediator.
type serverMediator struct {
	server             *server
	currentRoute       string
	defaultMiddlewares []func(http.Handler) http.Handler
	defaultHandler     http.Handler
	routesMiddlewares  map[string][]func(http.Handler) http.Handler
	routesHandler      map[string]http.Handler
	mu                 sync.RWMutex
}

// newServerMediator creates a new server mediator.
func newServerMediator(server *server) *serverMediator {
	return &serverMediator{
		server: server,
	}
}

// Name returns the server name.
func (m *serverMediator) Name() string {
	return m.server.name
}

// Listeners returns the server listeners.
func (m *serverMediator) Listeners() []string {
	return m.server.state.listeners
}

// Hosts returns the server hosts.
func (m *serverMediator) Hosts() []string {
	return m.server.state.hosts
}

// Store returns the server store.
func (m *serverMediator) Store() core.Store {
	return m.server.store
}

// Fetcher returns the server fetcher.
func (m *serverMediator) Fetcher() core.Fetcher {
	return m.server.fetcher
}

// RegisterMiddleware registers a middleware.
func (m *serverMediator) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentRoute == serverRouteDefault {
		m.defaultMiddlewares = append(m.defaultMiddlewares, middleware)
	} else {
		if m.routesMiddlewares == nil {
			m.routesMiddlewares = make(map[string][]func(http.Handler) http.Handler)
		}
		if middlewares, ok := m.routesMiddlewares[m.currentRoute]; ok {
			m.routesMiddlewares[m.currentRoute] = append(middlewares, middleware)
		} else {
			m.routesMiddlewares[m.currentRoute] = []func(http.Handler) http.Handler{middleware}
		}
	}

	return nil
}

// RegisterHandler registers a handler.
func (m *serverMediator) RegisterHandler(handler http.Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentRoute == serverRouteDefault {
		m.defaultHandler = handler
	} else {
		if m.routesHandler == nil {
			m.routesHandler = make(map[string]http.Handler)
		}
		if h, ok := m.routesHandler[m.currentRoute]; ok {
			m.routesHandler[m.currentRoute] = h
		} else {
			m.routesHandler[m.currentRoute] = handler
		}
	}

	return nil
}

var _ core.Server = (*serverMediator)(nil)

// serverRouter implements the server router.
type serverRouter struct {
	routes map[string]http.Handler
	mu     sync.RWMutex
}

// newServerRouter creates a new server router.
func newServerRouter() *serverRouter {
	return &serverRouter{
		routes: make(map[string]http.Handler),
	}
}

// AddRoute adds a route to the router.
func (r *serverRouter) addRoute(pattern string, handler http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[pattern] = handler
}

// Routes returns the router routes.
func (r *serverRouter) Routes() map[string]http.Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.routes
}

var _ ServerRouter = (*serverRouter)(nil)

// serverMiddleware implements the server middleware.
type serverMiddleware struct {
	server *server
	logger *log.Logger
}

const (
	serverMiddlewareHeaderRequestId string = "X-Request-ID"
	serverMiddlewareHeaderServer    string = "Server"

	serverMiddlewareHeaderServerValue string = "neon"
)

// newServerMiddleware creates the server middleware.
func newServerMiddleware(s *server) *serverMiddleware {
	return &serverMiddleware{
		server: s,
		logger: log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverLogger, s.name), log.LstdFlags|log.Lmsgprefix),
	}
}

// Handler implements the middleware handler.
func (m *serverMiddleware) Handler(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				if core.DEBUG {
					m.logger.Printf("%s, %s", err, debug.Stack())
				}
			}
		}()

		w.Header().Set(serverMiddlewareHeaderServer, serverMiddlewareHeaderServerValue)
		w.Header().Set(serverMiddlewareHeaderRequestId, uuid.NewString())

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

// serverHandler implements the default server handler.
type serverHandler struct {
	server *server
	logger *log.Logger
}

// newServerHandler creates the default server handler.
func newServerHandler(s *server) *serverHandler {
	return &serverHandler{
		server: s,
		logger: log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverLogger, s.name), log.LstdFlags|log.Lmsgprefix),
	}
}

// ServeHTTP implements the http handler.
func (h *serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
