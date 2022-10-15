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
	config     *RobotsRendererConfig
	logger     *log.Logger
	bufferPool BufferPool
	cache      Cache
	next       Renderer
}

// RobotsRendererConfig implements the robots renderer configuration
type RobotsRendererConfig struct {
	Path     string
	Hosts    []string
	Sitemaps []string
	Cache    bool
	CacheTTL int
}

const (
	robotsLogger string = "server[robots]"
)

// CreateRobotsRenderer creates a new robots renderer
func CreateRobotsRenderer(config *RobotsRendererConfig) (*robotsRenderer, error) {
	return &robotsRenderer{
		config:     config,
		logger:     log.New(os.Stderr, fmt.Sprint(robotsLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		bufferPool: newBufferPool(),
		cache:      newCache(),
	}, nil
}

// Handle implements the renderer
func (r *robotsRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	if req.URL.Path != r.config.Path {
		r.next.Handle(w, req, info)

		return
	}

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
func (r *robotsRenderer) Next(renderer Renderer) {
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

	body, err := robots(r, req)
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
		r.cache.Set(req.URL.Path, &result, time.Duration(r.config.CacheTTL)*time.Second)
		result.Cache = true
	}

	return &result, nil
}

// robots generates the robots.txt content
func robots(r *robotsRenderer, req *http.Request) ([]byte, error) {
	body := r.bufferPool.Get()
	defer r.bufferPool.Put(body)

	var check bool
	for _, host := range r.config.Hosts {
		if host == req.Host {
			check = true
		}
	}
	if !check {
		body.WriteString("User-agent: *\n")
		body.WriteString("Disallow: /\n")

		return body.Bytes(), nil
	}

	body.WriteString("User-agent: *\n")
	body.WriteString("Allow: /\n")

	for i, s := range r.config.Sitemaps {
		if i == 0 {
			body.WriteString("\n")
		}
		body.WriteString(fmt.Sprintf("Sitemap: %s\n", s))
	}

	return body.Bytes(), nil
}
