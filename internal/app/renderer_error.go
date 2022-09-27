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
	Renderer
	next Renderer

	config *ErrorRendererConfig
	logger *log.Logger
}

// ErrorRendererConfig implements the error renderer configuration
type ErrorRendererConfig struct {
}

const (
	ERROR_LOGGER string = "renderer[error]"
)

// CreateErrorRenderer creates a new error renderer
func CreateErrorRenderer(config *ErrorRendererConfig) (*errorRenderer, error) {
	logger := log.New(os.Stdout, fmt.Sprint(ERROR_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	return &errorRenderer{
		config: config,
		logger: logger,
	}, nil
}

// handle implements the renderer handler
func (r *errorRenderer) handle(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte{})
}

// setNext configures the next renderer
func (r *errorRenderer) setNext(renderer Renderer) {
	r.next = renderer
}
