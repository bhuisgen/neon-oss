package neon

import (
	"errors"
	"fmt"
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
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// serverSite implements a server site.
type serverSite struct {
	name   string
	config *serverSiteConfig
	logger *slog.Logger
	state  *serverSiteState
	server Server
	mu     sync.RWMutex
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
	store       core.Store
	server      core.Server
	mediator    *serverSiteMediator
	middleware  *serverSiteMiddleware
	handler     *serverSiteHandler
	router      *serverSiteRouter
}

// serverSiteRouteState implements a server site route state.
type serverSiteRouteState struct {
	middlewares map[string]core.ServerSiteMiddlewareModule
	handler     core.ServerSiteHandlerModule
}

const (
	serverSiteRouteDefault string = "default"
)

// newServerSite creates a new site.
func newServerSite(name string, server Server) *serverSite {
	return &serverSite{
		name:   name,
		logger: slog.New(log.NewHandler(os.Stderr, "app.server.site", nil)).With("name", name),
		server: server,
		state: &serverSiteState{
			routesMap: make(map[string]serverSiteRouteState),
		},
	}
}

// Init initializes the site.
func (s *serverSite) Init(config map[string]interface{}) error {
	s.logger.Debug("Initializing site")

	if config == nil {
		s.logger.Error("Missing configuration")
		return errors.New("missing configuration")
	}

	if err := mapstructure.Decode(config, &s.config); err != nil {
		s.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %w", err)
	}

	var errConfig bool

	if len(s.config.Listeners) == 0 {
		s.logger.Error("No listener defined")
		errConfig = true
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
			moduleInfo, err := module.Lookup(module.ModuleID("app.server.site.middleware." + middleware))
			if err != nil {
				s.logger.Error("Unregistered middleware module", "middleware", middleware, "err", err)
				errConfig = true
				continue
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteMiddlewareModule)
			if !ok {
				s.logger.Error("Invalid middleware module", "middleware", middleware, "err", err)
				errConfig = true
				continue
			}
			if middlewareConfig == nil {
				middlewareConfig = map[string]interface{}{}
			}
			if err := module.Init(middlewareConfig); err != nil {
				s.logger.Error("Failed to init middleware module", "middleware", middleware, "err", err)
				errConfig = true
				continue
			}

			stateRoute.middlewares[middleware] = module
		}

		for handler, handlerConfig := range routeConfig.Handler {
			moduleInfo, err := module.Lookup(module.ModuleID("app.server.site.handler." + handler))
			if err != nil {
				s.logger.Error("Unregistered handler module", "handler", handler, "err", err)
				errConfig = true
				break
			}
			module, ok := moduleInfo.NewInstance().(core.ServerSiteHandlerModule)
			if !ok {
				s.logger.Error("Invalid handler module", "handler", handler, "err", err)
				errConfig = true
				break
			}
			if handlerConfig == nil {
				handlerConfig = map[string]interface{}{}
			}
			if err := module.Init(handlerConfig); err != nil {
				s.logger.Error("Failed to init handler module", "handler", handler, "err", err)
				errConfig = true
				break
			}

			stateRoute.handler = module

			break
		}

		s.state.routes = append(s.state.routes, route)
		s.state.routesMap[route] = stateRoute
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the site.
func (s *serverSite) Register(app core.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering site")

	s.state.store = app.Store()
	s.state.server = app.Server()

	mediator := newServerSiteMediator(s, app)
	for _, route := range s.state.routes {
		mediator.currentRoute = route

		for _, middleware := range s.state.routesMap[route].middlewares {
			if err := middleware.Register(mediator); err != nil {
				return fmt.Errorf("register middleware: %w", err)
			}
		}
		if s.state.routesMap[route].handler != nil {
			if err := s.state.routesMap[route].handler.Register(mediator); err != nil {
				return fmt.Errorf("register handler: %w", err)
			}
		}
	}
	s.state.mediator = mediator
	s.state.middleware = newServerSiteMiddleware(s)
	s.state.handler = newServerSiteHandler(s)

	router, err := s.buildRouter()
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}
	s.state.router = router

	return nil
}

// Start starts the site.
func (s *serverSite) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting site")

	for _, route := range s.state.routes {
		for _, middleware := range s.state.routesMap[route].middlewares {
			if err := middleware.Start(); err != nil {
				return fmt.Errorf("start middleware: %w", err)
			}
		}
		if s.state.routesMap[route].handler != nil {
			if err := s.state.routesMap[route].handler.Start(); err != nil {
				return fmt.Errorf("start handler: %w", err)
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
			if err := middleware.Stop(); err != nil {
				return fmt.Errorf("stop middleware: %w", err)
			}
		}
		if s.state.routesMap[route].handler != nil {
			if err := s.state.routesMap[route].handler.Stop(); err != nil {
				return fmt.Errorf("stop handler: %w", err)
			}
		}
	}

	return nil
}

// Name returns the site name.
func (s *serverSite) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	app                core.App
	currentRoute       string
	defaultMiddlewares []func(http.Handler) http.Handler
	defaultHandler     http.Handler
	routesMiddlewares  map[string][]func(http.Handler) http.Handler
	routesHandler      map[string]http.Handler
	mu                 sync.RWMutex
}

// newServerSiteMediator creates a new mediator.
func newServerSiteMediator(site *serverSite, app core.App) *serverSiteMediator {
	return &serverSiteMediator{
		site: site,
		app:  app,
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

// Returns the store.
func (m *serverSiteMediator) Store() core.Store {
	return m.site.state.store
}

// Returns the server.
func (m *serverSiteMediator) Server() core.Server {
	return m.site.state.server
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
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				if !DEBUG {
					m.logger.Error("Error handler", "err", err)
				} else {
					m.logger.Error("Error handler", "err", err, "stack", string(debug.Stack()))
				}
			}
		}()

		w.Header().Set(serverSiteMiddlewareHeaderServer, serverSiteMiddlewareHeaderServerValue)
		w.Header().Set(serverSiteMiddlewareHeaderRequestId, uuid.NewString())

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
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
