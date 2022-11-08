// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"io"
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

// robotsRender implements a render
type robotsRender struct {
	Body   []byte
	Status int
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

	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*robotsRender)
			w.WriteHeader(result.Status)
			w.Write(result.Body)

			r.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", req.URL.Path, result.Status, true)

			return
		}
	}

	b := r.bufferPool.Get()
	defer r.bufferPool.Put(b)

	err := r.render(req, b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, http.StatusInternalServerError)

		return
	}

	if r.config.Cache {
		body := make([]byte, b.Len())
		copy(body, b.Bytes())

		r.cache.Set(req.URL.Path, &robotsRender{
			Body:   body,
			Status: http.StatusOK,
		}, time.Duration(r.config.CacheTTL)*time.Second)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b.Bytes())

	r.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", req.URL.Path, http.StatusOK, false)
}

// Next configures the next renderer
func (r *robotsRenderer) Next(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *robotsRenderer) render(req *http.Request, w io.Writer) error {
	var check bool
	for _, host := range r.config.Hosts {
		if host == req.Host {
			check = true
		}
	}
	if !check {
		w.Write([]byte("User-agent: *\n"))
		w.Write([]byte("Disallow: /\n"))

		return nil
	}

	w.Write([]byte("User-agent: *\n"))
	w.Write([]byte("Allow: /\n"))

	for i, s := range r.config.Sitemaps {
		if i == 0 {
			w.Write([]byte("\n"))
		}
		w.Write([]byte(fmt.Sprintf("Sitemap: %s\n", s)))
	}

	return nil
}
