// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/mitchellh/mapstructure"
)

// server implements the server.
type server struct {
	config  *serverConfig
	logger  *log.Logger
	state   *serverState
	store   Store
	fetcher Fetcher
	loader  Loader
}

// serverConfig implements the server configuration.
type serverConfig struct {
	Listeners map[string]map[string]interface{}
	Sites     map[string]map[string]interface{}
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
		logger: log.New(os.Stderr, fmt.Sprint(serverLogger, ": "), log.LstdFlags|log.Lmsgprefix),
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
	if config == nil {
		s.logger.Print("missing configuration")
		return errors.New("missing configuration")
	}

	if err := mapstructure.Decode(config, &s.config); err != nil {
		s.logger.Print("failed to parse configuration")
		return err
	}

	var errInit bool

	if len(s.config.Listeners) == 0 {
		s.logger.Print("no listener defined")
		errInit = true
	}
	for listenerName, listenerConfig := range s.config.Listeners {
		listener := newServerListener(listenerName)

		if listenerConfig == nil {
			listenerConfig = map[string]interface{}{}
		}
		if err := listener.Init(
			listenerConfig,
			log.New(os.Stderr, fmt.Sprint(s.logger.Prefix(), "listener[", listenerName, "]: "), log.LstdFlags|log.Lmsgprefix),
		); err != nil {
			s.logger.Printf("failed to init listener '%s'", listenerName)
			errInit = true
			continue
		}

		s.state.listenersMap[listenerName] = listener
	}

	if len(s.config.Sites) == 0 {
		s.logger.Print("no site defined")
		errInit = true
	}
	for siteName, siteConfig := range s.config.Sites {
		site := newServerSite(siteName, s.store, s.fetcher, s.loader, s)

		if siteConfig == nil {
			siteConfig = map[string]interface{}{}
		}
		if err := site.Init(
			siteConfig,
			log.New(os.Stderr, fmt.Sprint(s.logger.Prefix(), "site[", siteName, "]: "), log.LstdFlags|log.Lmsgprefix),
		); err != nil {
			s.logger.Printf("failed to init site '%s'", siteName)
			errInit = true
			continue
		}

		s.state.sitesMap[siteName] = site
	}

	if errInit {
		return errors.New("init error")
	}

	return nil
}

// Register registers the server listeners descriptors.
func (s *server) Register(descriptors map[string]ServerListenerDescriptor) error {
	for listenerName, listener := range s.state.listenersMap {
		if err := listener.Register(descriptors[listenerName]); err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Register(); err != nil {
			return err
		}
	}

	return nil
}

// Start starts the server.
func (s *server) Start() error {
	for _, listener := range s.state.listenersMap {
		if err := listener.Serve(); err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Start(); err != nil {
			return err
		}

		for listenerName, listener := range s.state.listenersMap {
			listener.Link(site)
			s.state.sitesListeners[site.Name()] = append(s.state.sitesListeners[site.Name()],
				s.state.listenersMap[listenerName])
		}
	}

	return nil
}

// Stop stops the server.
func (s *server) Stop() error {
	for _, site := range s.state.sitesMap {
		if err := site.Stop(); err != nil {
			return err
		}
	}

	for _, listener := range s.state.listenersMap {
		if err := listener.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Shutdown shutdowns the server gracefully.
func (s *server) Shutdown(ctx context.Context) error {
	for _, listener := range s.state.listenersMap {
		if err := listener.Shutdown(ctx); err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		if err := site.Stop(); err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		listeners, ok := s.state.sitesListeners[site.Name()]
		if !ok {
			s.logger.Print("site is not linked")
			continue
		}
		for _, listener := range listeners {
			listener.Unlink(site)
		}
		delete(s.state.sitesListeners, site.Name())
	}

	for _, listener := range s.state.listenersMap {
		if err := listener.Close(); err != nil {
			return err
		}
		if err := listener.Remove(); err != nil {
			return err
		}
	}

	return nil
}

var _ (Server) = (*server)(nil)
