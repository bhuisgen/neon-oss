package neon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
)

// server implements the server.
type server struct {
	config *serverConfig
	logger *slog.Logger
	state  *serverState
}

// serverConfig implements the server configuration.
type serverConfig struct {
	Listeners map[string]map[string]interface{} `mapstructure:"listeners"`
	Sites     map[string]map[string]interface{} `mapstructure:"sites"`
}

// serverState implements the server state.
type serverState struct {
	listenersMap   map[string]ServerListener
	sitesMap       map[string]ServerSite
	sitesListeners map[string][]ServerListener
	mediator       *serverMediator
}

const (
	serverModuleID module.ModuleID = "app.server"
)

// ModuleInfo returns the module information.
func (s server) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: serverModuleID,
		NewInstance: func() module.Module {
			return &server{
				logger: slog.New(log.NewHandler(os.Stderr, string(serverModuleID), nil)),
				state: &serverState{
					listenersMap:   make(map[string]ServerListener),
					sitesMap:       make(map[string]ServerSite),
					sitesListeners: make(map[string][]ServerListener),
				},
			}
		},
	}
}

// Init initializes the server.
func (s *server) Init(config map[string]interface{}) error {
	s.logger.Debug("Initializing server")

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
	for listenerName, listenerConfig := range s.config.Listeners {
		listener := newServerListener(listenerName, s)

		if listenerConfig == nil {
			listenerConfig = map[string]interface{}{}
		}
		if err := listener.Init(
			listenerConfig,
		); err != nil {
			s.logger.Error("Failed to init listener", "name", listenerName, "err", err)
			errConfig = true
			continue
		}

		s.state.listenersMap[listenerName] = listener
	}

	if len(s.config.Sites) == 0 {
		s.logger.Error("No site defined")
		errConfig = true
	}
	var defaultSiteName string
	for siteName, siteConfig := range s.config.Sites {
		site := newServerSite(siteName, s)

		if siteConfig == nil {
			siteConfig = map[string]interface{}{}
		}

		if err := site.Init(
			siteConfig,
		); err != nil {
			s.logger.Error("Failed to init site", "site", siteName, "err", err)
			errConfig = true
			continue
		}
		if site.Default() && defaultSiteName != "" {
			err := fmt.Errorf("default site already defined: %s", defaultSiteName)
			s.logger.Error("Failed to init site", "site", siteName, "err", err)
			errConfig = true
		}
		defaultSiteName = site.Name()

		s.state.sitesMap[siteName] = site
	}

	if errConfig {
		return errors.New("config")
	}

	return nil
}

// Register registers the server.
func (s *server) Register(app core.App) error {
	s.logger.Debug("Registering server")

	s.state.mediator = newServerMediator(s)

	for listenerName, listener := range s.state.listenersMap {
		if err := listener.Register(app); err != nil {
			return fmt.Errorf("register listener %s: %w", listenerName, err)
		}
	}

	for siteName, site := range s.state.sitesMap {
		if err := site.Register(app); err != nil {
			return fmt.Errorf("register site %s: %w", siteName, err)
		}
	}

	return nil
}

// Start starts the server.
func (s *server) Start() error {
	s.logger.Info("Starting server")

	for _, site := range s.state.sitesMap {
		for listenerName, listener := range s.state.listenersMap {
			if err := listener.Link(site); err != nil {
				return fmt.Errorf("link site: %w", err)
			}
			s.state.sitesListeners[site.Name()] = append(s.state.sitesListeners[site.Name()],
				s.state.listenersMap[listenerName])
		}
	}
	for _, listener := range s.state.listenersMap {
		if err := listener.Serve(); err != nil {
			return fmt.Errorf("serve listener: %w", err)
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Start(); err != nil {
			return fmt.Errorf("start site: %w", err)
		}
	}

	return nil
}

// Stop stops the server.
func (s *server) Stop() error {
	s.logger.Info("Stopping server")

	for _, listener := range s.state.listenersMap {
		if err := listener.Close(); err != nil {
			return fmt.Errorf("close listener: %w", err)
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Stop(); err != nil {
			return fmt.Errorf("stop site: %w", err)
		}
	}

	return nil
}

// Shutdown shutdowns the server gracefully.
func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")

	for _, listener := range s.state.listenersMap {
		if err := listener.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown listener: %w", err)
		}
	}
	for _, site := range s.state.sitesMap {
		if listeners, ok := s.state.sitesListeners[site.Name()]; ok {
			for _, listener := range listeners {
				if err := listener.Unlink(site); err != nil {
					return fmt.Errorf("unlink listener: %w", err)
				}
			}
			delete(s.state.sitesListeners, site.Name())
		} else {
			s.logger.Warn("Site was not linked", "site", site.Name())
		}
	}
	for _, listener := range s.state.listenersMap {
		if err := listener.Close(); err != nil {
			return fmt.Errorf("close listener: %w", err)
		}
		if err := listener.Remove(); err != nil {
			return fmt.Errorf("remove listener: %w", err)
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Stop(); err != nil {
			return fmt.Errorf("stop site: %w", err)
		}
	}

	return nil
}

// Listeners returns the network listeners.
func (s *server) Listeners() (map[string][]net.Listener, error) {
	m := make(map[string][]net.Listener, len(s.state.listenersMap))
	for name, listener := range s.state.listenersMap {
		listeners, err := listener.Listeners()
		if err != nil {
			return nil, fmt.Errorf("get listeners: %w", err)
		}
		m[name] = listeners
	}
	return m, nil
}

var _ Server = (*server)(nil)

// serverMediator implements the server mediator.
type serverMediator struct {
	server *server
}

// newServerMediator creates a new mediator.
func newServerMediator(server *server) *serverMediator {
	return &serverMediator{
		server: server,
	}
}

var _ core.Server = (*serverMediator)(nil)
