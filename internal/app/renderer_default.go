// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// defaultRenderer implements the default renderer
type defaultRenderer struct {
	config     *DefaultRendererConfig
	logger     *log.Logger
	cache      Cache
	next       Renderer
	osReadFile func(name string) ([]byte, error)
}

// DefaultRendererConfig implements the default renderer configuration
type DefaultRendererConfig struct {
	File       string
	StatusCode int
	Cache      bool
	CacheTTL   int
}

const (
	defaultLogger string = "server[default]"
)

// defaultOsReadFile redirects to os.ReadFile
func defaultOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// CreateDefaultRenderer creates a new default renderer
func CreateDefaultRenderer(config *DefaultRendererConfig) (*defaultRenderer, error) {
	return &defaultRenderer{
		config:     config,
		logger:     log.New(os.Stderr, fmt.Sprint(defaultLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		cache:      newCache(),
		osReadFile: defaultOsReadFile,
	}, nil
}

// Handle implements the renderer
func (r *defaultRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	result, err := r.render(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, http.StatusInternalServerError)

		return
	}

	w.WriteHeader(result.Status)
	w.Write(result.Body)

	r.logger.Printf("Render completed (url=%s, status=%d, valid=%t, cache=%t)", req.URL.Path, result.Status, result.Valid,
		result.Cache)
}

// Next configures the next renderer
func (r *defaultRenderer) Next(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *defaultRenderer) render(req *http.Request) (*Render, error) {
	if r.config.Cache {
		obj := r.cache.Get("default")
		if obj != nil {
			result := obj.(*Render)

			return result, nil
		}
	}

	body, err := defaultFile(r, req)
	if err != nil {
		r.logger.Printf("Failed to render: %s", err)

		return nil, err
	}

	result := Render{
		Body:   body,
		Valid:  true,
		Status: http.StatusOK,
	}
	if result.Valid && r.config.Cache {
		r.cache.Set("default", &result, time.Duration(r.config.CacheTTL)*time.Second)
		result.Cache = true
	}

	return &result, nil
}

// defaultFile generates a default file
func defaultFile(r *defaultRenderer, req *http.Request) ([]byte, error) {
	body, err := r.osReadFile(r.config.File)
	if err != nil {
		r.logger.Printf("Failed to read default file '%s': %s", r.config.File, err)

		return nil, err
	}

	return body, nil
}
