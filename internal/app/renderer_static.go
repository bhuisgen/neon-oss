// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// staticRenderer implements the static renderer
type staticRenderer struct {
	Renderer
	next Renderer

	config        *StaticRendererConfig
	logger        *log.Logger
	staticFS      *staticFileSystem
	staticHandler http.Handler
}

// StaticRendererConfig implements the static renderer configuration
type StaticRendererConfig struct {
	Enable bool
	Dir    string
	Index  bool
}

const (
	STATIC_LOGGER string = "renderer[static]"
)

// CreateStaticRenderer creates a new static renderer
func CreateStaticRenderer(config *StaticRendererConfig) (*staticRenderer, error) {
	logger := log.New(os.Stdout, fmt.Sprint(STATIC_LOGGER, ": "), log.LstdFlags|log.Lmsgprefix)

	dir, err := filepath.Abs(config.Dir)
	if err != nil {
		return nil, err
	}

	staticFS := &staticFileSystem{
		FileSystem: http.Dir(dir),
		prefix:     dir,
		index:      config.Index,
	}
	staticHandler := http.FileServer(staticFS)

	return &staticRenderer{
		config:        config,
		logger:        logger,
		staticFS:      staticFS,
		staticHandler: staticHandler,
	}, nil
}

// handle implements the renderer handler
func (r *staticRenderer) handle(w http.ResponseWriter, req *http.Request) {
	if !r.staticFS.exists(req.URL.Path) {
		r.next.handle(w, req)

		return
	}

	r.staticHandler.ServeHTTP(w, req)
}

// setNext configures the next renderer
func (r *staticRenderer) setNext(renderer Renderer) {
	r.next = renderer
}

// staticFileSystem implements a static filesystem
type staticFileSystem struct {
	http.FileSystem

	prefix string
	index  bool
}

// exists checks if a file exists into the fileystem
func (fs *staticFileSystem) exists(name string) bool {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	name = filepath.Join(fs.prefix, name)

	s, err := os.Stat(name)
	if err != nil {
		return false
	}

	if s.IsDir() {
		if !fs.index {
			return false
		}

		name = filepath.Join(name, "index.html")

		_, err = os.Stat(name)
	}

	return err == nil
}
