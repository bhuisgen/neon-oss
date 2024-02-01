package neon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mitchellh/mapstructure"
)

// server implements the server.
type server struct {
	config  *serverConfig
	logger  *slog.Logger
	state   *serverState
	store   Store
	fetcher Fetcher
	loader  Loader
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
}

const (
	serverLogger string = "server"
)

// newServer creates a new server.
func newServer(store Store, fetcher Fetcher, loader Loader) *server {
	return &server{
		logger: slog.New(NewLogHandler(os.Stderr, serverLogger, nil)),
		state: &serverState{
			listenersMap:   make(map[string]ServerListener),
			sitesMap:       make(map[string]ServerSite),
			sitesListeners: make(map[string][]ServerListener),
		},
		store:   store,
		fetcher: fetcher,
		loader:  loader,
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
		listener := newServerListener(listenerName)

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
		site := newServerSite(siteName, s.store, s.fetcher, s.loader, s)

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
			err := fmt.Errorf("default site already defined (%s)", defaultSiteName)
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

// Register registers the server listeners and sites.
func (s *server) Register(descriptors map[string]ServerListenerDescriptor) error {
	s.logger.Debug("Registering server")

	for listenerName, listener := range s.state.listenersMap {
		if err := listener.Register(descriptors[listenerName]); err != nil {
			return fmt.Errorf("register listener: %w", err)
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Register(); err != nil {
			return fmt.Errorf("register site: %w", err)
		}
	}

	return nil
}

// Start starts the server.
func (s *server) Start() error {
	s.logger.Info("Starting server")

	for _, listener := range s.state.listenersMap {
		if err := listener.Serve(); err != nil {
			return fmt.Errorf("serve listener: %w", err)
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Start(); err != nil {
			return fmt.Errorf("start site: %w", err)
		}

		for listenerName, listener := range s.state.listenersMap {
			if err := listener.Link(site); err != nil {
				return fmt.Errorf("link site: %w", err)
			}
			s.state.sitesListeners[site.Name()] = append(s.state.sitesListeners[site.Name()],
				s.state.listenersMap[listenerName])
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
		listeners, ok := s.state.sitesListeners[site.Name()]
		if !ok {
			s.logger.Warn("Site is not linked")
			continue
		}
		for _, listener := range listeners {
			if err := listener.Unlink(site); err != nil {
				return fmt.Errorf("unlink listener: %w", err)
			}
		}
		delete(s.state.sitesListeners, site.Name())
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

var _ (Server) = (*server)(nil)
