// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
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

// serverSite implements a server site.
type serverSite struct {
	name    string
	config  *serverSiteConfig
	logger  *log.Logger
	state   *serverSiteState
	store   Store
	fetcher Fetcher
	loader  Loader
	server  Server
	mu      sync.RWMutex
}

// serverSiteConfig implements the server site configuration.
type serverSiteConfig struct {
	Listeners []string
	Hosts     []string
	Routes    map[string]serverSiteRouteConfig
}

// serverSiteRouteConfig implements a server site route configuration.
type serverSiteRouteConfig struct {
	Middlewares map[string]interface{}
	Handler     map[string]interface{}
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
	middlewares        []string
	middlewaresModules map[string]core.ServerSiteMiddlewareModule
	handler            string
	handlerModule      core.ServerSiteHandlerModule
}

const (
	serverSiteLogger       string = "site"
	serverSiteRouteDefault string = "default"
)

// newServerSite creates a new site.
func newServerSite(name string, store Store, fetcher Fetcher, loader Loader, server Server) *serverSite {
	return &serverSite{
		name:    name,
		store:   store,
		fetcher: fetcher,
		loader:  loader,
		server:  server,
	}
}

// Check checks the site configuration.
func (s *serverSite) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c serverSiteConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "site: failed to parse configuration")
		return report, err
	}

	if len(c.Listeners) == 0 {
		report = append(report, "site: no listener defined")
	}
	for _, routeConfig := range c.Routes {
		for middleware, middlewareConfig := range routeConfig.Middlewares {
			moduleInfo, err := module.Lookup(module.ModuleID("server.site.middleware." + middleware))
			if err != nil {
				report = append(report, fmt.Sprintf("site: unregistered middleware module '%s'", middleware))
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteMiddlewareModule)
			if !ok {
				report = append(report, fmt.Sprintf("site: invalid middleware module '%s'", middleware))
				continue
			}
			var moduleConfig map[string]interface{}
			moduleConfig, _ = middlewareConfig.(map[string]interface{})
			r, err := module.Check(moduleConfig)
			if err != nil {
				for _, line := range r {
					report = append(report, fmt.Sprintf("site: middleware '%s', failed to check configuration: %s", middleware,
						line))
					continue
				}
			}
		}

		for handler, handlerConfig := range routeConfig.Handler {
			moduleInfo, err := module.Lookup(module.ModuleID("server.site.handler." + handler))
			if err != nil {
				report = append(report, fmt.Sprintf("site: unregistered handler module '%s'", handler))
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteHandlerModule)
			if !ok {
				report = append(report, fmt.Sprintf("site: invalid handler module '%s'", handler))
				continue
			}
			var moduleConfig map[string]interface{}
			moduleConfig, _ = handlerConfig.(map[string]interface{})
			r, err := module.Check(moduleConfig)
			if err != nil {
				for _, line := range r {
					report = append(report, fmt.Sprintf("site: handler '%s', failed to check configuration: %s", handler, line))
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

// Load loads the site.
func (s *serverSite) Load(config map[string]interface{}) error {
	var c serverSiteConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	s.config = &c
	s.logger = log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverSiteLogger, s.name), log.LstdFlags|log.Lmsgprefix)
	s.state = &serverSiteState{
		routesMap: make(map[string]serverSiteRouteState),
	}

	s.state.hosts = append(s.state.hosts, s.config.Hosts...)
	s.state.listeners = append(s.state.listeners, s.config.Listeners...)

	for route, routeConfig := range s.config.Routes {
		stateRoute := serverSiteRouteState{
			middlewaresModules: make(map[string]core.ServerSiteMiddlewareModule),
		}

		for middleware, middlewareConfig := range routeConfig.Middlewares {
			moduleInfo, err := module.Lookup(module.ModuleID("server.site.middleware." + middleware))
			if err != nil {
				return err
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteMiddlewareModule)
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
			moduleInfo, err := module.Lookup(module.ModuleID("server.site.handler." + handler))
			if err != nil {
				return err
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteHandlerModule)
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

// Register registers the site middlewares and handlers.
func (s *serverSite) Register() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mediator := newServerSiteMediator(s)

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
	s.state.middleware = newServerSiteMiddleware(s)
	s.state.handler = newServerSiteHandler(s)

	return nil
}

// Start starts the site.
func (s *serverSite) Start() error {
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

// Stop stops the site.
func (s *serverSite) Stop() error {
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

	router := newServerSiteRouter()

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
	routes map[string]http.Handler
	mu     sync.RWMutex
}

// newServerSiteRouter creates a new server site router.
func newServerSiteRouter() *serverSiteRouter {
	return &serverSiteRouter{
		routes: make(map[string]http.Handler),
	}
}

// AddRoute adds a route to the router.
func (r *serverSiteRouter) addRoute(pattern string, handler http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[pattern] = handler
}

// Routes returns the router routes.
func (r *serverSiteRouter) Routes() map[string]http.Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.routes
}

var _ ServerSiteRouter = (*serverSiteRouter)(nil)

// serverSiteMiddleware implements the server site middleware.
type serverSiteMiddleware struct {
	site   *serverSite
	logger *log.Logger
}

const (
	serverSiteMiddlewareHeaderRequestId string = "X-Request-ID"
	serverSiteMiddlewareHeaderServer    string = "Server"

	serverSiteMiddlewareHeaderServerValue string = "neon"
)

// newServerSiteMiddleware creates the server site middleware.
func newServerSiteMiddleware(s *serverSite) *serverSiteMiddleware {
	return &serverSiteMiddleware{
		site:   s,
		logger: log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverSiteLogger, s.name), log.LstdFlags|log.Lmsgprefix),
	}
}

// Handler implements the middleware handler.
func (m *serverSiteMiddleware) Handler(next http.Handler) http.Handler {
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

		w.Header().Set(serverSiteMiddlewareHeaderServer, serverSiteMiddlewareHeaderServerValue)
		w.Header().Set(serverSiteMiddlewareHeaderRequestId, uuid.NewString())

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

// serverSiteHandler implements the default server site handler.
type serverSiteHandler struct {
	site   *serverSite
	logger *log.Logger
}

// newServerSiteHandler creates the site handler.
func newServerSiteHandler(s *serverSite) *serverSiteHandler {
	return &serverSiteHandler{
		site:   s,
		logger: log.New(os.Stderr, fmt.Sprintf("%s[%s]: ", serverSiteLogger, s.name), log.LstdFlags|log.Lmsgprefix),
	}
}

// ServeHTTP implements the http handler.
func (h *serverSiteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
