// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/bhuisgen/neon/internal/app/middlewares"
)

// server implements a server
type server struct {
	config                      *ServerConfig
	logger                      *log.Logger
	httpServer                  *http.Server
	renderer                    Renderer
	info                        *ServerInfo
	osCreate                    func(name string) (*os.File, error)
	osReadFile                  func(name string) ([]byte, error)
	httpServerListenAndServe    func(server *http.Server) error
	httpServerListenAndServeTLS func(server *http.Server, certFile string, keyFile string) error
	httpServerShutdown          func(server *http.Server, context context.Context) error
}

// ServerConfig implements the server configuration
type ServerConfig struct {
	ListenAddr    string
	ListenPort    int
	TLS           bool
	TLSCAFile     *string
	TLSCertFile   *string
	TLSKeyFile    *string
	ReadTimeout   int
	WriteTimeout  int
	AccessLog     bool
	AccessLogFile *string
	Renderer      *ServerRendererConfig
}

// ServerRendererConfig implements the server renderers configuration
type ServerRendererConfig struct {
	Rewrite *RewriteRendererConfig
	Header  *HeaderRendererConfig
	Static  *StaticRendererConfig
	Robots  *RobotsRendererConfig
	Sitemap *SitemapRendererConfig
	Index   *IndexRendererConfig
	Default *DefaultRendererConfig
}

type ContextKeyID struct{}

const (
	serverLogger string = "server"
)

// serverOsCreate redirects to os.Create
func serverOsCreate(name string) (*os.File, error) {
	return os.Create(name)
}

// serverOsReadFile redirects to os.ReadFile
func serverOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// serverHttpServerListenAndServe redirects to http.Server.ListenAndServe
func serverHttpServerListenAndServe(server *http.Server) error {
	return server.ListenAndServe()
}

// serverHttpListenAndServeTLS redirects to http.Server.ListenAndServeTLS
func serverHttpListenAndServeTLS(server *http.Server, certFile string, keyFile string) error {
	return server.ListenAndServeTLS(certFile, keyFile)
}

// httpServerShutdown redirects to http.Server.Shutdown
func httpServerShutdown(server *http.Server, context context.Context) error {
	return server.Shutdown(context)
}

// CreateServer creates a new server instance
func CreateServer(config *ServerConfig, renderers ...Renderer) (*server, error) {
	s := server{
		config:   config,
		logger:   log.New(os.Stderr, fmt.Sprint(serverLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		renderer: nil,
		info: &ServerInfo{
			Addr:    config.ListenAddr,
			Port:    config.ListenPort,
			Version: Version,
		},
		osCreate:                    serverOsCreate,
		osReadFile:                  serverOsReadFile,
		httpServerListenAndServe:    serverHttpServerListenAndServe,
		httpServerListenAndServeTLS: serverHttpListenAndServeTLS,
		httpServerShutdown:          httpServerShutdown,
	}

	err := s.initialize(renderers...)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// initialize initializes the server
func (s *server) initialize(renderers ...Renderer) error {
	var previous Renderer
	for index, renderer := range renderers {
		if index == 0 {
			s.renderer = renderer
			previous = renderer
			continue
		}

		previous.Next(renderer)
		previous = renderer
	}

	recoverConfig := middlewares.RecoverConfig{}

	loggerConfig := middlewares.LoggerConfig{
		Log: s.config.AccessLog,
	}
	if s.config.AccessLogFile != nil {
		logFile, err := s.osCreate(*s.config.AccessLogFile)
		if err != nil {
			return err
		}
		loggerConfig.Writer = logFile
	}

	mux := http.NewServeMux()
	mux.Handle("/", middlewares.Recover(&recoverConfig, middlewares.Logger(&loggerConfig, NewServerHandler(s))))

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.ListenAddr, s.config.ListenPort),
		Handler:      mux,
		ReadTimeout:  time.Duration(s.config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.WriteTimeout) * time.Second,
	}

	if s.config.TLS {
		tlsConfig := &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			MinVersion: tls.VersionTLS12,
		}

		if s.config.TLSCAFile != nil {
			ca, err := s.osReadFile(*s.config.TLSCAFile)
			if err != nil {
				return err
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(ca)

			tlsConfig.ClientCAs = caCertPool
		}

		s.httpServer.TLSConfig = tlsConfig
	}

	return nil
}

// Start starts the server instance
func (s *server) Start() error {
	go func() {
		if s.config.TLS {
			s.logger.Printf("Listening at https://%s", s.httpServer.Addr)

			err := s.httpServerListenAndServeTLS(s.httpServer, *s.config.TLSCertFile, *s.config.TLSKeyFile)
			if err != nil && err != http.ErrServerClosed {
				log.Print(err)
			}
		} else {
			s.logger.Printf("Listening at http://%s", s.httpServer.Addr)

			err := s.httpServerListenAndServe(s.httpServer)
			if err != nil && err != http.ErrServerClosed {
				log.Print(err)
			}
		}
	}()

	return nil
}

// Stop stops the server instance
func (s *server) Stop(ctx context.Context) error {
	err := s.httpServerShutdown(s.httpServer, ctx)
	if err != nil {
		return err
	}

	return nil
}

// serverHandler implements the server handler
type serverHandler struct {
	server *server
}

// NewServerHandler creates a new server handler
func NewServerHandler(server *server) *serverHandler {
	return &serverHandler{
		server: server,
	}
}

// ServeHTTP implements the HTTP server handler
func (h *serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := uuid.New()

	ctx := context.WithValue(r.Context(), ContextKeyID{}, id.String())
	r = r.WithContext(ctx)

	w.Header().Set("Server", fmt.Sprint("neon/", Version))
	w.Header().Set("X-Correlation-ID", id.String())

	h.server.renderer.Handle(w, r, h.server.info)
}
