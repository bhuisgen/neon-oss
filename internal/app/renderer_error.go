// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"log"
	"net/http"
)

// errorRenderer implements the error renderer
type errorRenderer struct {
	Renderer
	next Renderer

	config *ErrorRendererConfig
	logger *log.Logger
}

// ErrorRendererConfig implements the error renderer configuration
type ErrorRendererConfig struct {
	StatusCode int
}

// CreateErrorRenderer creates a new error renderer
func CreateErrorRenderer(config *ErrorRendererConfig) (*errorRenderer, error) {
	return &errorRenderer{
		config: config,
		logger: log.Default(),
	}, nil
}

// handle implements the renderer handler
func (r *errorRenderer) handle(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(r.config.StatusCode)
	w.Write([]byte{})
}

// setNext configures the next renderer
func (r *errorRenderer) setNext(renderer Renderer) {
	r.next = renderer
}
