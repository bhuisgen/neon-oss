// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// staticRenderer implements the static renderer
type staticRenderer struct {
	config        *StaticRendererConfig
	logger        *log.Logger
	staticFS      StaticFileSystem
	staticHandler http.Handler
	next          Renderer
}

// StaticRendererConfig implements the static renderer configuration
type StaticRendererConfig struct {
	Dir   string
	Index bool
}

const (
	staticLogger string = "server[static]"
)

// CreateStaticRenderer creates a new static renderer
func CreateStaticRenderer(config *StaticRendererConfig) (*staticRenderer, error) {
	r := staticRenderer{
		config: config,
		logger: log.New(os.Stderr, fmt.Sprint(staticLogger, ": "), log.LstdFlags|log.Lmsgprefix),
	}

	err := r.initialize()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// initialize initializes the renderer
func (r *staticRenderer) initialize() error {
	dir, err := filepath.Abs(r.config.Dir)
	if err != nil {
		return err
	}

	r.staticFS = &staticFileSystem{
		FileSystem: http.Dir(dir),
		prefix:     dir,
		index:      r.config.Index,
		osStat:     staticFileSystemOsStat,
	}
	r.staticHandler = http.FileServer(r.staticFS)

	return nil
}

// Handle implements the renderer
func (r *staticRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
	if !r.staticFS.Exists(req.URL.Path) {
		r.next.Handle(w, req, i)

		return
	}

	r.staticHandler.ServeHTTP(w, req)
}

// Next configures the next renderer
func (r *staticRenderer) Next(renderer Renderer) {
	r.next = renderer
}

// StaticFileSystem
type StaticFileSystem interface {
	Exists(name string) bool
	Open(name string) (http.File, error)
}

// staticFileSystem implements the default static filesystem
type staticFileSystem struct {
	http.FileSystem
	prefix string
	index  bool
	osStat func(name string) (fs.FileInfo, error)
}

// staticFileSystemOsStat redirects to os.Stat
func staticFileSystemOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// Exists checks if a file or an index exists
func (fs *staticFileSystem) Exists(name string) bool {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	name = filepath.Join(fs.prefix, name)
	s, err := fs.osStat(name)
	if err != nil {
		return false
	}

	if s.IsDir() {
		if !fs.index {
			return false
		}
		name = filepath.Join(name, "index.html")
		_, err = fs.osStat(name)
	}

	return err == nil
}

// Open implements FileSystem using os.Open, opening files for reading rooted
// and relative to the directory d.
func (fs *staticFileSystem) Open(name string) (http.File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("invalid character in file path")
	}
	dir := fs.prefix
	if dir == "" {
		dir = "."
	}
	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	f, err := os.Open(fullName)
	if err != nil {
		return nil, err
	}
	return f, nil
}
