// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
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
	Sitemaps []string
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
	var body = bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(body)
	body.Reset()

	body.WriteString("User-Agent: *\n")

	var check bool
	for _, host := range r.config.Hosts {
		if host == req.Host {
			check = true
		}
	}
	if !check {
		body.WriteString("Disallow: /\n")

		return body.Bytes(), nil
	}

	body.WriteString("Allow: /\n")

	for _, s := range r.config.Sitemaps {
		body.WriteString(fmt.Sprintf("Sitemap: %s\n", s))
	}

	return body.Bytes(), nil
}
