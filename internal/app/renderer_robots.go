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

// robotsRenderer implements the robots renderer
type robotsRenderer struct {
	Renderer
	next Renderer

	config *RobotsRendererConfig
	logger *log.Logger
	cache  *cache
}

// RobotsRendererConfig implements the robots renderer configuration
type RobotsRendererConfig struct {
	Enable   bool
	Path     string
	Hosts    []string
	Cache    bool
	CacheTTL int
}

const (
	robotsLogger string = "server[robots]"
)

// CreateRobotsRenderer creates a new robots renderer
func CreateRobotsRenderer(config *RobotsRendererConfig, loader *loader) (*robotsRenderer, error) {
	logger := log.New(os.Stdout, fmt.Sprint(robotsLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	return &robotsRenderer{
		config: config,
		logger: logger,
		cache:  NewCache(),
	}, nil
}

// handle implements the renderer handler
func (r *robotsRenderer) handle(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != r.config.Path {
		r.next.handle(w, req)

		return
	}

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

// setNext configures the next renderer
func (r *robotsRenderer) setNext(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *robotsRenderer) render(req *http.Request) (*Render, error) {
	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*Render)

			return result, nil
		}
	}

	var body []byte

	var allow bool
	for _, host := range r.config.Hosts {
		if host == req.Host {
			allow = true
		}
	}

	if allow {
		body = []byte("User-Agent: *\nAllow: /\n")
	} else {
		body = []byte("User-Agent: *\nDisallow: /\n")
	}

	result := Render{
		Body:   body,
		Status: http.StatusOK,
		Valid:  true,
	}
	if result.Valid && r.config.Cache {
		r.cache.Set(req.URL.Path, &result, time.Duration(r.config.CacheTTL)*time.Second)
		result.Cache = true
	}

	return &result, nil
}
