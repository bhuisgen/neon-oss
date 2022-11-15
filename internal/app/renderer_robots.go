// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"
)

// robotsRenderer implements the robots renderer
type robotsRenderer struct {
	config     *RobotsRendererConfig
	logger     *log.Logger
	template   *template.Template
	bufferPool BufferPool
	cache      Cache
	next       Renderer
}

// RobotsRendererConfig implements the robots renderer configuration
type RobotsRendererConfig struct {
	Path     string
	Hosts    []string
	Cache    bool
	CacheTTL int
	Sitemaps []string
}

// robotsRender implements a render
type robotsRender struct {
	Body   []byte
	Status int
}

// robotsTemplateData implements the robots template data
type robotsTemplateData struct {
	Check    bool
	Sitemaps []string
}

const (
	robotsLogger string = "server[robots]"
)

var (
	//go:embed templates/robots/robots.txt.tmpl
	robotsTemplate string
)

// CreateRobotsRenderer creates a new robots renderer
func CreateRobotsRenderer(config *RobotsRendererConfig) (*robotsRenderer, error) {
	template, err := template.New("robots").Parse(robotsTemplate)
	if err != nil {
		return nil, err
	}

	return &robotsRenderer{
		config:     config,
		logger:     log.New(os.Stderr, fmt.Sprint(robotsLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		template:   template,
		bufferPool: newBufferPool(),
		cache:      newCache(),
	}, nil
}

// Handle implements the renderer
func (r *robotsRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
	if req.URL.Path != r.config.Path {
		r.next.Handle(w, req, i)

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

	err := r.template.Execute(w, robotsTemplateData{
		Check:    check,
		Sitemaps: r.config.Sitemaps,
	})
	if err != nil {
		return err
	}

	return nil
}
