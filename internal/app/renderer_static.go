// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"log"
	"net/http"
	"os"
	"path"
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
	Path   string
}

// CreateStaticRenderer creates a new static renderer
func CreateStaticRenderer(config *StaticRendererConfig) (*staticRenderer, error) {
	static, err := filepath.Abs(config.Path)
	if err != nil {
		return nil, err
	}

	staticFS := &staticFileSystem{
		FileSystem: http.Dir(static),
		root:       static,
		index:      "index.html",
	}
	staticHandler := http.FileServer(staticFS)

	return &staticRenderer{
		config:        config,
		logger:        log.Default(),
		staticFS:      staticFS,
		staticHandler: staticHandler,
	}, nil
}

// handle implements the renderer handler
func (r *staticRenderer) handle(w http.ResponseWriter, req *http.Request) {
	if !r.config.Enable {
		r.next.handle(w, req)

		return
	}

	if !r.staticFS.exists("/", req.URL.Path) {
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
	root  string
	index string
}

// exists checks if a file exists into the fileystem
func (f *staticFileSystem) exists(prefix string, urlpath string) bool {
	filepath := strings.TrimPrefix(urlpath, prefix)
	if filepath == urlpath {
		return false
	}

	realpath := path.Join(f.root, filepath)

	stat, err := os.Stat(realpath)
	if err != nil {
		return false
	}

	if stat.IsDir() {
		if f.index == "" {
			return false
		}

		indexpath := path.Join(realpath, f.index)

		_, err = os.Stat(indexpath)
	}

	return err == nil
}
