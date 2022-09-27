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
	Rewrite       RewriteRendererConfig
	Header        HeaderRendererConfig
	Static        StaticRendererConfig
	Robots        RobotsRendererConfig
	Sitemap       SitemapRendererConfig
	Index         IndexRendererConfig
	Default       DefaultRendererConfig
}

type ContextKeyID struct{}

const (
	SERVER_LOGGER string = "server"
)

// CreateServer creates a new instance
func CreateServer(config *ServerConfig, renderers ...Renderer) (*Server, error) {
	logger := log.New(os.Stdout, fmt.Sprint(SERVER_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	server := Server{
		config:   config,
		logger:   logger,
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
			s.logger.Printf("Listening at https://%s", s.httpServer.Addr)

			err := s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
			if err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		} else {
			s.logger.Printf("Listening at http://%s", s.httpServer.Addr)

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
	id := uuid.New()

	ctx := context.WithValue(r.Context(), ContextKeyID{}, id.String())
	r = r.WithContext(ctx)

	w.Header().Set("X-Correlation-ID", id.String())

	h.server.renderer.handle(w, r)
}
