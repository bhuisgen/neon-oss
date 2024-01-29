// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"errors"
	"log/slog"
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

// serverSite implements a server site.
type serverSite struct {
	name    string
	config  *serverSiteConfig
	logger  *slog.Logger
	state   *serverSiteState
	store   Store
	fetcher Fetcher
	loader  Loader
	server  Server
	mu      sync.RWMutex
}

// serverSiteConfig implements the server site configuration.
type serverSiteConfig struct {
	Listeners []string                         `mapstructure:"listeners"`
	Hosts     []string                         `mapstructure:"hosts"`
	Routes    map[string]serverSiteRouteConfig `mapstructure:"routes"`
}

// serverSiteRouteConfig implements a server site route configuration.
type serverSiteRouteConfig struct {
	Middlewares map[string]map[string]interface{} `mapstructure:"middlewares"`
	Handler     map[string]map[string]interface{} `mapstructure:"handler"`
}

// serverSiteState implements the server site state.
type serverSiteState struct {
	listeners   []string
	hosts       []string
	defaultSite bool
	routes      []string
	routesMap   map[string]serverSiteRouteState
	mediator    *serverSiteMediator
	router      *serverSiteRouter
	middleware  *serverSiteMiddleware
	handler     *serverSiteHandler
}

// serverSiteRouteState implements a server site route state.
type serverSiteRouteState struct {
	middlewares map[string]core.ServerSiteMiddlewareModule
	handler     core.ServerSiteHandlerModule
}

const (
	serverSiteLogger       string = "site"
	serverSiteRouteDefault string = "default"
)

// newServerSite creates a new site.
func newServerSite(name string, store Store, fetcher Fetcher, loader Loader,
	server Server) *serverSite {
	return &serverSite{
		name:   name,
		logger: slog.New(NewLogHandler(os.Stderr, serverSiteLogger, nil)).With("name", name),
		state: &serverSiteState{
			routesMap: make(map[string]serverSiteRouteState),
		},
		store:   store,
		fetcher: fetcher,
		loader:  loader,
		server:  server,
	}
}

// Init initializes the site.
func (s *serverSite) Init(config map[string]interface{}) error {
	if config == nil {
		s.logger.Error("Missing configuration")
		return errors.New("missing configuration")
	}

	if err := mapstructure.Decode(config, &s.config); err != nil {
		s.logger.Error("Failed to parse configuration")
		return err
	}

	var errInit bool

	if len(s.config.Listeners) == 0 {
		s.logger.Error("No listener defined")
		errInit = true
	}

	s.state.listeners = append(s.state.listeners, s.config.Listeners...)
	s.state.hosts = append(s.state.hosts, s.config.Hosts...)
	if len(s.state.hosts) == 0 {
		s.state.defaultSite = true
	}

	for route, routeConfig := range s.config.Routes {
		stateRoute := serverSiteRouteState{
			middlewares: make(map[string]core.ServerSiteMiddlewareModule),
		}

		for middleware, middlewareConfig := range routeConfig.Middlewares {
			moduleInfo, err := module.Lookup(module.ModuleID("server.site.middleware." + middleware))
			if err != nil {
				s.logger.Error("Unregistered middleware module", "middleware", middleware)
				errInit = true
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteMiddlewareModule)
			if !ok {
				s.logger.Error("Invalid middleware module", "middleware", middleware)
				errInit = true
				continue
			}
			if middlewareConfig == nil {
				middlewareConfig = map[string]interface{}{}
			}
			if err := module.Init(
				middlewareConfig,
				slog.New(NewLogHandler(os.Stderr, serverSiteLogger, nil)).With("middleware", middleware),
			); err != nil {
				s.logger.Error("Failed to init middleware module", "middleware", middleware)
				errInit = true
				continue
			}

			stateRoute.middlewares[middleware] = module
		}

		for handler, handlerConfig := range routeConfig.Handler {
			moduleInfo, err := module.Lookup(module.ModuleID("server.site.handler." + handler))
			if err != nil {
				s.logger.Error("Unregistered handler module", "handler", handler)
				errInit = true
				break
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteHandlerModule)
			if !ok {
				s.logger.Error("Invalid handler module", "handler", handler)
				errInit = true
				break
			}
			if handlerConfig == nil {
				handlerConfig = map[string]interface{}{}
			}
			if err := module.Init(
				handlerConfig,
				slog.New(NewLogHandler(os.Stderr, serverSiteLogger, nil)).With("handler", handler),
			); err != nil {
				s.logger.Error("Failed to init handler module", "handler", handler)
				errInit = true
				break
			}

			stateRoute.handler = module

			break
		}

		s.state.routes = append(s.state.routes, route)
		s.state.routesMap[route] = stateRoute
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Register registers the site middlewares and handlers.
func (s *serverSite) Register() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering site")

	mediator := newServerSiteMediator(s)

	for _, route := range s.state.routes {
		mediator.currentRoute = route

		for _, middleware := range s.state.routesMap[route].middlewares {
			if err := middleware.Register(mediator); err != nil {
				return err
			}
		}
		if s.state.routesMap[route].handler != nil {
			if err := s.state.routesMap[route].handler.Register(mediator); err != nil {
				return err
			}
		}
	}

	s.state.mediator = mediator

	router, err := s.buildRouter()
	if err != nil {
		return err
	}

	s.state.router = router
	s.state.middleware = newServerSiteMiddleware(s)
	s.state.handler = newServerSiteHandler(s)

	return nil
}

// Start starts the site.
func (s *serverSite) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting site")

	for _, route := range s.state.routes {
		for _, middleware := range s.state.routesMap[route].middlewares {
			err := middleware.Start()
			if err != nil {
				return err
			}
		}
		if s.state.routesMap[route].handler != nil {
			if err := s.state.routesMap[route].handler.Start(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Stop stops the site.
func (s *serverSite) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Stopping site")

	for _, route := range s.state.routes {
		for _, middleware := range s.state.routesMap[route].middlewares {
			middleware.Stop()
		}
		if s.state.routesMap[route].handler != nil {
			s.state.routesMap[route].handler.Stop()
		}
	}

	return nil
}

// Name returns the site name.
func (s *serverSite) Name() string {
	return s.name
}

// Listeners returns the site listeners.
func (s *serverSite) Listeners() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.listeners
}

// Hosts returns the site hosts.
func (s *serverSite) Hosts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.hosts
}

// Default returns true if the site is the default site.
func (s *serverSite) Default() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.defaultSite
}

// Router returns the site router.
func (s *serverSite) Router() (ServerSiteRouter, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state.router == nil {
		return nil, errors.New("server not ready")
	}

	return s.state.router, nil
}

// buildRouter builds the site router.
func (s *serverSite) buildRouter() (*serverSiteRouter, error) {
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

	router := newServerSiteRouter(s)

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

var _ ServerSite = (*serverSite)(nil)

// serverSiteMediator implements the server site mediator.
type serverSiteMediator struct {
	site               *serverSite
	currentRoute       string
	defaultMiddlewares []func(http.Handler) http.Handler
	defaultHandler     http.Handler
	routesMiddlewares  map[string][]func(http.Handler) http.Handler
	routesHandler      map[string]http.Handler
	mu                 sync.RWMutex
}

// newServerSiteMediator creates a new server site mediator.
func newServerSiteMediator(site *serverSite) *serverSiteMediator {
	return &serverSiteMediator{
		site: site,
	}
}

// Name returns the site name.
func (m *serverSiteMediator) Name() string {
	return m.site.name
}

// Listeners returns the site listeners.
func (m *serverSiteMediator) Listeners() []string {
	return m.site.state.listeners
}

// Hosts returns the site hosts.
func (m *serverSiteMediator) Hosts() []string {
	return m.site.state.hosts
}

// Store returns the store.
func (m *serverSiteMediator) Store() core.Store {
	return m.site.store.(core.Store)
}

// Fetcher returns the fetcher.
func (m *serverSiteMediator) Fetcher() core.Fetcher {
	return m.site.fetcher.(core.Fetcher)
}

// Loader returns the loader.
func (m *serverSiteMediator) Loader() core.Loader {
	return m.site.loader.(core.Loader)
}

// Server returns the server.
func (m *serverSiteMediator) Server() core.Server {
	return m.site.server.(core.Server)
}

// RegisterMiddleware registers a middleware.
func (m *serverSiteMediator) RegisterMiddleware(middleware func(next http.Handler) http.Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentRoute == serverSiteRouteDefault {
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
func (m *serverSiteMediator) RegisterHandler(handler http.Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentRoute == serverSiteRouteDefault {
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

var _ core.ServerSite = (*serverSiteMediator)(nil)

// serverSiteRouter implements the server site router.
type serverSiteRouter struct {
	logger *slog.Logger
	routes map[string]http.Handler
}

// newServerSiteRouter creates a new server site router.
func newServerSiteRouter(s *serverSite) *serverSiteRouter {
	return &serverSiteRouter{
		logger: s.logger,
		routes: make(map[string]http.Handler),
	}
}

// AddRoute adds a route to the router.
func (r *serverSiteRouter) addRoute(pattern string, handler http.Handler) {
	r.routes[pattern] = handler
}

// Routes returns the router routes.
func (r *serverSiteRouter) Routes() map[string]http.Handler {
	return r.routes
}

var _ ServerSiteRouter = (*serverSiteRouter)(nil)

// serverSiteMiddleware implements the server site middleware.
type serverSiteMiddleware struct {
	logger *slog.Logger
}

const (
	serverSiteMiddlewareHeaderRequestId string = "X-Request-ID"
	serverSiteMiddlewareHeaderServer    string = "Server"

	serverSiteMiddlewareHeaderServerValue string = "neon"
)

// newServerSiteMiddleware creates the server site middleware.
func newServerSiteMiddleware(s *serverSite) *serverSiteMiddleware {
	return &serverSiteMiddleware{
		logger: s.logger,
	}
}

// Handler implements the middleware handler.
func (m *serverSiteMiddleware) Handler(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				m.logger.Error("Error handler", "err", err, "stack", debug.Stack())
			}
		}()

		w.Header().Set(serverSiteMiddlewareHeaderServer, serverSiteMiddlewareHeaderServerValue)
		w.Header().Set(serverSiteMiddlewareHeaderRequestId, uuid.NewString())

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

// serverSiteHandler implements the default server site handler.
type serverSiteHandler struct {
	logger *slog.Logger
}

// newServerSiteHandler creates the site handler.
func newServerSiteHandler(s *serverSite) *serverSiteHandler {
	return &serverSiteHandler{
		logger: s.logger,
	}
}

// ServeHTTP implements the http handler.
func (h *serverSiteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Error("No handler available")

	http.NotFound(w, r)
}
