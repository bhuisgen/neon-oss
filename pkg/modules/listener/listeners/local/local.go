// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package local

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// localListener implements the local listener.
type localListener struct {
	config             *localListenerConfig
	logger             *log.Logger
	listener           net.Listener
	server             *http.Server
	osReadFile         func(name string) ([]byte, error)
	netListen          func(network string, addr string) (net.Listener, error)
	httpServerServe    func(server *http.Server, listener net.Listener) error
	httpServerShutdown func(server *http.Server, context context.Context) error
	httpServerClose    func(server *http.Server) error
}

// localListenerConfig implements the local listener configuration.
type localListenerConfig struct {
	ListenAddr        *string
	ListenPort        *int
	ReadTimeout       *int
	ReadHeaderTimeout *int
	WriteTimeout      *int
	IdleTimeout       *int
}

const (
	localModuleID module.ModuleID = "listener.local"
	localLogger   string          = "listener.local"

	localConfigDefaultListenAddr        string = ""
	localConfigDefaultListenPort        int    = 80
	localConfigDefaultReadTimeout       int    = 60
	localConfigDefaultReadHeaderTimeout int    = 10
	localConfigDefaultWriteTimeout      int    = 60
	localConfigDefaultIdleTimeout       int    = 60
)

// localOsReadFile redirects to os.ReadFile.
func localOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// localNetListen redirects to net.Listen.
func localNetListen(network string, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}

// localHttpServerServe redirects to http.Server.Serve.
func localHttpServerServe(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

// localHttpServerShutdown redirects to http.Server.Shutdown.
func localHttpServerShutdown(server *http.Server, context context.Context) error {
	return server.Shutdown(context)
}

// localHttpServerShutdown redirects to http.Server.Close.
func localHttpServerClose(server *http.Server) error {
	return server.Close()
}

// init initializes the module.
func init() {
	module.Register(localListener{})
}

// ModuleInfo returns the module information.
func (l localListener) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: localModuleID,
		NewInstance: func() module.Module {
			return &localListener{
				osReadFile:         localOsReadFile,
				netListen:          localNetListen,
				httpServerServe:    localHttpServerServe,
				httpServerShutdown: localHttpServerShutdown,
				httpServerClose:    localHttpServerClose,
			}
		},
	}
}

// Check checks the listener configuration.
func (l *localListener) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c localListenerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.ListenAddr == nil {
		defaultValue := localConfigDefaultListenAddr
		c.ListenAddr = &defaultValue
	}
	if c.ListenPort == nil {
		defaultValue := localConfigDefaultListenPort
		c.ListenPort = &defaultValue
	}
	if *c.ListenPort < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ListenPort", *c.ListenPort))
	}
	if c.ReadTimeout == nil {
		defaultValue := localConfigDefaultReadTimeout
		c.ReadTimeout = &defaultValue
	}
	if *c.ReadTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ReadTimeout", *c.ReadTimeout))
	}
	if c.ReadHeaderTimeout == nil {
		defaultValue := localConfigDefaultReadHeaderTimeout
		c.ReadHeaderTimeout = &defaultValue
	}
	if *c.ReadHeaderTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ReadHeaderTimeout", *c.ReadHeaderTimeout))
	}
	if c.WriteTimeout == nil {
		defaultValue := localConfigDefaultWriteTimeout
		c.WriteTimeout = &defaultValue
	}
	if *c.WriteTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "WriteTimeout", *c.WriteTimeout))
	}
	if c.IdleTimeout == nil {
		defaultValue := localConfigDefaultIdleTimeout
		c.IdleTimeout = &defaultValue
	}
	if *c.IdleTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "IdleTimeout", *c.IdleTimeout))
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the listener.
func (l *localListener) Load(config map[string]interface{}) error {
	var c localListenerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	l.config = &c
	l.logger = log.New(os.Stderr, fmt.Sprint(localLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	if l.config.ListenAddr == nil {
		defaultValue := localConfigDefaultListenAddr
		l.config.ListenAddr = &defaultValue
	}
	if l.config.ListenPort == nil {
		defaultValue := localConfigDefaultListenPort
		l.config.ListenPort = &defaultValue
	}
	if l.config.ReadTimeout == nil {
		defaultValue := localConfigDefaultReadTimeout
		l.config.ReadTimeout = &defaultValue
	}
	if l.config.ReadHeaderTimeout == nil {
		defaultValue := localConfigDefaultReadHeaderTimeout
		l.config.ReadHeaderTimeout = &defaultValue
	}
	if l.config.WriteTimeout == nil {
		defaultValue := localConfigDefaultWriteTimeout
		l.config.WriteTimeout = &defaultValue
	}
	if l.config.IdleTimeout == nil {
		defaultValue := localConfigDefaultIdleTimeout
		l.config.IdleTimeout = &defaultValue
	}

	return nil
}

// Register registers the listener.
func (l *localListener) Register(listener core.Listener) error {
	if len(listener.Listeners()) == 1 {
		l.listener = listener.Listeners()[0]
		return nil
	}

	var err error
	l.listener, err = l.netListen("tcp", fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort))
	if err != nil {
		return err
	}

	err = listener.RegisterListener(l.listener)
	if err != nil {
		return err
	}

	return nil
}

// Serve accepts incoming connections.
func (l *localListener) Serve(handler http.Handler) error {
	l.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort),
		Handler:           handler,
		ReadTimeout:       time.Duration(*l.config.ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(*l.config.ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(*l.config.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(*l.config.IdleTimeout) * time.Second,
		ErrorLog:          l.logger,
	}

	go func() {
		l.logger.Printf("Listening at http://%s", l.server.Addr)

		err := l.httpServerServe(l.server, l.listener)
		if err != nil && err != http.ErrServerClosed {
			log.Print(err)
		}
	}()

	return nil
}

// Shutdown shutdowns the listener gracefully.
func (l *localListener) Shutdown(ctx context.Context) error {
	err := l.httpServerShutdown(l.server, ctx)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the listener.
func (l *localListener) Close() error {
	err := l.httpServerClose(l.server)
	if err != nil {
		return err
	}

	return nil
}

var _ core.ListenerModule = (*localListener)(nil)
