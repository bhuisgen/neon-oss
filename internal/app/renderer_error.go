// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// errorRenderer implements the error renderer
type errorRenderer struct {
	config *ErrorRendererConfig
	logger *log.Logger
}

// ErrorRendererConfig implements the error renderer configuration
type ErrorRendererConfig struct {
}

const (
	errorLogger string = "server[error]"
)

// CreateErrorRenderer creates a new error renderer
func CreateErrorRenderer(config *ErrorRendererConfig) (*errorRenderer, error) {
	return &errorRenderer{
		config: config,
		logger: log.New(os.Stderr, fmt.Sprint(errorLogger, ": "), log.LstdFlags|log.Lmsgprefix),
	}, nil
}

// Handle implements the renderer
func (r *errorRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte{})

	r.logger.Printf("Render completed (url=%s, status=%d)", req.URL.Path, http.StatusInternalServerError)
}

// Next configures the next renderer
func (r *errorRenderer) Next(renderer Renderer) {
}
