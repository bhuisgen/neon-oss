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
		store:   store,
		fetcher: fetcher,
		loader:  loader,
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
	for listenerName, listenerConfig := range c.Listeners {
		listener := newServerListener(listenerName)

		r, err := listener.Check(listenerConfig)
		if err != nil {
			for _, line := range r {
				report = append(report, fmt.Sprintf("server: listener '%s', failed to check configuration: %s", listenerName,
					line))
			}
			continue
		}
	}

	if len(c.Sites) == 0 {
		report = append(report, "server: no site defined")
	}
	for siteName, siteConfig := range c.Sites {
		site := newServerSite(siteName, s.store, s.fetcher, s.loader, s)

		r, err := site.Check(siteConfig)
		if err != nil {
			for _, line := range r {
				report = append(report, fmt.Sprintf("server: site '%s', failed to check configuration: %s", siteName, line))
			}
			continue
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
	s.logger = log.New(os.Stderr, fmt.Sprint(serverLogger, ": "), log.LstdFlags|log.Lmsgprefix)
	s.state = &serverState{
		listenersMap:   make(map[string]ServerListener),
		sitesMap:       make(map[string]ServerSite),
		sitesListeners: make(map[string][]ServerListener),
	}

	for listenerName, listenerConfig := range c.Listeners {
		listener := newServerListener(listenerName)

		err := listener.Load(listenerConfig)
		if err != nil {
			return err
		}

		s.state.listenersMap[listenerName] = listener
	}

	for siteName, siteConfig := range c.Sites {
		site := newServerSite(siteName, s.store, s.fetcher, s.loader, s)

		err := site.Load(siteConfig)
		if err != nil {
			return err
		}

		s.state.sitesMap[siteName] = site
	}

	return nil
}

// Register registers the server listeners descriptors.
func (s *server) Register(descriptors map[string]ServerListenerDescriptor) error {
	for listenerName, listener := range s.state.listenersMap {
		err := listener.Register(descriptors[listenerName])
		if err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		err := site.Register()
		if err != nil {
			return err
		}
	}

	return nil
}

// Start starts the server.
func (s *server) Start() error {
	for _, listener := range s.state.listenersMap {
		err := listener.Serve()
		if err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		err := site.Start()
		if err != nil {
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
		err := site.Stop()
		if err != nil {
			return err
		}
	}

	for _, listener := range s.state.listenersMap {
		err := listener.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// Shutdown shutdowns the server gracefully.
func (s *server) Shutdown(ctx context.Context) error {
	for _, listener := range s.state.listenersMap {
		err := listener.Shutdown(ctx)
		if err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		err := site.Stop()
		if err != nil {
			return err
		}
	}

	for _, site := range s.state.sitesMap {
		listeners, ok := s.state.sitesListeners[site.Name()]
		if !ok {
			return errors.New("site not linked")
		}

		for _, listener := range listeners {
			listener.Unlink(site)
		}
		delete(s.state.sitesListeners, site.Name())
	}

	for _, listener := range s.state.listenersMap {
		err := listener.Close()
		if err != nil {
			return err
		}

		err = listener.Remove()
		if err != nil {
			return err
		}
	}

	return nil
}

var _ (Server) = (*server)(nil)
