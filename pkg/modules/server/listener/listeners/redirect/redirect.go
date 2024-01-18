// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package redirect

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
)

// redirectListener implements the redirect listener.
type redirectListener struct {
	config             *redirectListenerConfig
	logger             *log.Logger
	listener           net.Listener
	server             *http.Server
	osReadFile         func(name string) ([]byte, error)
	netListen          func(network string, addr string) (net.Listener, error)
	httpServerServe    func(server *http.Server, listener net.Listener) error
	httpServerShutdown func(server *http.Server, context context.Context) error
	httpServerClose    func(server *http.Server) error
}

// redirectListenerConfig implements the redirect listener configuration.
type redirectListenerConfig struct {
	ListenAddr        *string
	ListenPort        *int
	ReadTimeout       *int
	ReadHeaderTimeout *int
	WriteTimeout      *int
	IdleTimeout       *int
	RedirectPort      *int
}

const (
	redirectModuleID module.ModuleID = "server.listener.redirect"
	redirectLogger   string          = "listener[redirect]"

	redirectConfigDefaultListenAddr        string = ""
	redirectConfigDefaultListenPort        int    = 80
	redirectConfigDefaultReadTimeout       int    = 60
	redirectConfigDefaultReadHeaderTimeout int    = 10
	redirectConfigDefaultWriteTimeout      int    = 60
	redirectConfigDefaultIdleTimeout       int    = 60
)

// redirectOsReadFile redirects to os.ReadFile.
func redirectOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// redirectNetListen redirects to net.Listen.
func redirectNetListen(network string, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}

// redirectHttpServerServe redirects to http.Server.Serve.
func redirectHttpServerServe(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

// redirectHttpServerShutdown redirects to http.Server.Shutdown.
func redirectHttpServerShutdown(server *http.Server, context context.Context) error {
	return server.Shutdown(context)
}

// redirectHttpServerClose redirects to http.Server.Close.
func redirectHttpServerClose(server *http.Server) error {
	return server.Close()
}

// init initializes the module.
func init() {
	module.Register(redirectListener{})
}

// ModuleInfo returns the module information.
func (l redirectListener) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: redirectModuleID,
		NewInstance: func() module.Module {
			return &redirectListener{
				osReadFile:         redirectOsReadFile,
				netListen:          redirectNetListen,
				httpServerServe:    redirectHttpServerServe,
				httpServerShutdown: redirectHttpServerShutdown,
				httpServerClose:    redirectHttpServerClose,
			}
		},
	}
}

// Check checks the listener configuration.
func (l *redirectListener) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c redirectListenerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.ListenAddr == nil {
		defaultValue := redirectConfigDefaultListenAddr
		c.ListenAddr = &defaultValue
	}
	if c.ListenPort == nil {
		defaultValue := redirectConfigDefaultListenPort
		c.ListenPort = &defaultValue
	}
	if *c.ListenPort < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ListenPort", *c.ListenPort))
	}
	if c.ReadTimeout == nil {
		defaultValue := redirectConfigDefaultReadTimeout
		c.ReadTimeout = &defaultValue
	}
	if *c.ReadTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ReadTimeout", *c.ReadTimeout))
	}
	if c.ReadHeaderTimeout == nil {
		defaultValue := redirectConfigDefaultReadHeaderTimeout
		c.ReadHeaderTimeout = &defaultValue
	}
	if *c.ReadHeaderTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "ReadHeaderTimeout", *c.ReadHeaderTimeout))
	}
	if c.WriteTimeout == nil {
		defaultValue := redirectConfigDefaultWriteTimeout
		c.WriteTimeout = &defaultValue
	}
	if *c.WriteTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "WriteTimeout", *c.WriteTimeout))
	}
	if c.IdleTimeout == nil {
		defaultValue := redirectConfigDefaultIdleTimeout
		c.IdleTimeout = &defaultValue
	}
	if *c.IdleTimeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "IdleTimeout", *c.IdleTimeout))
	}
	if c.RedirectPort != nil && *c.RedirectPort < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "RedirectPort", *c.RedirectPort))
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the listener.
func (l *redirectListener) Load(config map[string]interface{}) error {
	var c redirectListenerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	l.config = &c
	l.logger = log.New(os.Stderr, fmt.Sprint(redirectLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	if l.config.ListenAddr == nil {
		defaultValue := redirectConfigDefaultListenAddr
		l.config.ListenAddr = &defaultValue
	}
	if l.config.ListenPort == nil {
		defaultValue := redirectConfigDefaultListenPort
		l.config.ListenPort = &defaultValue
	}
	if l.config.ReadTimeout == nil {
		defaultValue := redirectConfigDefaultReadTimeout
		l.config.ReadTimeout = &defaultValue
	}
	if l.config.ReadHeaderTimeout == nil {
		defaultValue := redirectConfigDefaultReadHeaderTimeout
		l.config.ReadHeaderTimeout = &defaultValue
	}
	if l.config.WriteTimeout == nil {
		defaultValue := redirectConfigDefaultWriteTimeout
		l.config.WriteTimeout = &defaultValue
	}
	if l.config.IdleTimeout == nil {
		defaultValue := redirectConfigDefaultIdleTimeout
		l.config.IdleTimeout = &defaultValue
	}

	return nil
}

// Register registers the listener.
func (l *redirectListener) Register(listener core.ServerListener) error {
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
func (l *redirectListener) Serve(handler http.Handler) error {
	l.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", *l.config.ListenAddr, *l.config.ListenPort),
		Handler:           http.HandlerFunc(l.redirectHandler),
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
func (l *redirectListener) Shutdown(ctx context.Context) error {
	err := l.httpServerShutdown(l.server, ctx)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the listener.
func (l *redirectListener) Close() error {
	err := l.httpServerClose(l.server)
	if err != nil {
		return err
	}

	return nil
}

// redirectHandler is the redirect handler.
func (l *redirectListener) redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Use HTTPS", http.StatusBadRequest)
		return
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	var target string
	if l.config.RedirectPort == nil {
		target = "https://" + host + r.URL.RequestURI()
	} else {
		target = "https://" + net.JoinHostPort(host, strconv.Itoa(*l.config.RedirectPort)) + r.URL.RequestURI()
	}

	http.Redirect(w, r, target, http.StatusFound)
}

var _ core.ServerListenerModule = (*redirectListener)(nil)
