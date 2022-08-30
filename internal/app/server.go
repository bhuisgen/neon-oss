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

	"github.com/bhuisgen/neon/internal/app/middlewares"
)

// server implements the server instance
type Server struct {
	config     *ServerConfig
	logger     *log.Logger
	httpServer *http.Server
	renderer   Renderer
}

// ServerConfig implements the server configuration
type ServerConfig struct {
	ListenAddr    string
	ListenPort    int
	TLS           bool
	TLSCAFile     string
	TLSCertFile   string
	TLSKeyFile    string
	ReadTimeout   int
	WriteTimeout  int
	AccessLog     bool
	AccessLogFile string
	ErrorCode     int
	Rewrite       RewriteRendererConfig
	Static        StaticRendererConfig
	Index         IndexRendererConfig
	Robots        RobotsRendererConfig
	Sitemap       SitemapRendererConfig
}

// CreateServer creates a new instance
func CreateServer(config *ServerConfig, renderers ...Renderer) (*Server, error) {
	server := Server{
		config:   config,
		logger:   log.Default(),
		renderer: nil,
	}

	var previous Renderer
	for index, renderer := range renderers {
		if index == 0 {
			server.renderer = renderer
			previous = renderer
			continue
		}

		previous.setNext(renderer)
		previous = renderer
	}

	mux := http.NewServeMux()
	mux.Handle("/", middlewares.Recover(middlewares.Logging(&middlewares.LoggingConfig{
		Log:     server.config.AccessLog,
		LogFile: server.config.AccessLogFile,
	}, NewServerHandler(&server))))

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.ListenAddr, config.ListenPort),
		Handler:      mux,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
	}

	if config.TLS {
		ca, err := os.ReadFile(config.TLSCAFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(ca)

		server.httpServer.TLSConfig = &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
			MinVersion: tls.VersionTLS12,
		}
	}

	return &server, nil
}

// Start starts the server instance
func (s *Server) Start() {
	go func() {
		if s.config.TLS {
			err := s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
			if err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		} else {
			err := s.httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}
	}()
}

// Stop stops the server instance
func (s *Server) Stop(ctx context.Context) {
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

// serverHandler implements the server handler
type serverHandler struct {
	server *Server
}

// NewServerHandler creates a new server handler
func NewServerHandler(server *Server) *serverHandler {
	return &serverHandler{
		server: server,
	}
}

// ServeHTTP implements the HTTP server handler
func (h *serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.renderer.handle(w, r)
}
