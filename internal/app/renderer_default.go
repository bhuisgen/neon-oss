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

// defaultRenderer implements the default renderer
type defaultRenderer struct {
	config     *DefaultRendererConfig
	logger     *log.Logger
	bufferPool BufferPool
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

// defaultRender implements a render
type defaultRender struct {
	Body   []byte
	Status int
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
		bufferPool: newBufferPool(),
		cache:      newCache(),
		osReadFile: defaultOsReadFile,
	}, nil
}

// Handle implements the renderer
func (r *defaultRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*defaultRender)
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

		r.cache.Set(req.URL.Path, &defaultRender{
			Body:   body,
			Status: http.StatusOK,
		}, time.Duration(r.config.CacheTTL)*time.Second)
	}

	w.WriteHeader(r.config.StatusCode)
	w.Write(b.Bytes())

	r.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", req.URL.Path, r.config.StatusCode, false)
}

// Next configures the next renderer
func (r *defaultRenderer) Next(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *defaultRenderer) render(req *http.Request, w io.Writer) error {
	body, err := r.osReadFile(r.config.File)
	if err != nil {
		r.logger.Printf("Failed to read default file '%s': %s", r.config.File, err)

		return err
	}

	w.Write(body)

	return nil
}
