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
	Render
	next Renderer

	config *DefaultRendererConfig
	logger *log.Logger
	cache  *cache
}

// DefaultRendererConfig implements the default renderer configuration
type DefaultRendererConfig struct {
	Enable     bool
	File       string
	StatusCode int
	Cache      bool
	CacheTTL   int
}

const (
	DEFAULT_LOGGER string = "renderer[default]"
)

// CreateDefaultRenderer creates a new default renderer
func CreateDefaultRenderer(config *DefaultRendererConfig) (*defaultRenderer, error) {
	logger := log.New(os.Stdout, fmt.Sprint(DEFAULT_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	return &defaultRenderer{
		config: config,
		logger: logger,
		cache:  NewCache(),
	}, nil
}

// handle implements the default renderer
func (r *defaultRenderer) handle(w http.ResponseWriter, req *http.Request) {
	result, err := r.render(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, result.Status)

		return
	}

	w.WriteHeader(result.Status)
	w.Write(result.Body)

	r.logger.Printf("Render completed (url=%s, status=%d, valid=%t, cache=%t)", req.URL.Path, result.Status, result.Valid,
		result.Cache)
}

// setNext configures the default renderer
func (r *defaultRenderer) setNext(renderer Renderer) {
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

	body, err := os.ReadFile(r.config.File)
	if err != nil {
		r.logger.Printf("Failed to read default file '%s': %s", r.config.File, err)

		return nil, err
	}

	result := Render{
		Body:   body,
		Status: http.StatusOK,
		Valid:  true,
	}
	if result.Valid && r.config.Cache {
		r.cache.Set("default", &result, time.Duration(r.config.CacheTTL)*time.Second)
		result.Cache = true
	}

	return &result, nil
}
