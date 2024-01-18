// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package file

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/cache/memory"
	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// fileHandler implements the html handler.
type fileHandler struct {
	config     *fileHandlerConfig
	logger     *log.Logger
	file       []byte
	fileInfo   *time.Time
	rwPool     render.RenderWriterPool
	cache      cache.Cache
	osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osReadFile func(name string) ([]byte, error)
	osClose    func(*os.File) error
	osStat     func(name string) (fs.FileInfo, error)
}

// fileHandlerConfig implements the default html configuration.
type fileHandlerConfig struct {
	Path       string
	StatusCode *int
	Cache      *bool
	CacheTTL   *int
}

const (
	fileModuleID module.ModuleID = "server.site.handler.file"
	fileLogger   string          = "handler[file]"

	fileConfigDefaultStatusCode int  = 200
	fileConfigDefaultCache      bool = false
	fileConfigDefaultCacheTTL   int  = 60
)

// fileOsOpenFile redirects to os.OpenFile.
func fileOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// fileOsReadFile redirects to os.ReadFile.
func fileOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// fileOsClose redirects to os.Close.
func fileOsClose(f *os.File) error {
	return f.Close()
}

// fileOsStat redirects to os.Stat.
func fileOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// init initializes the module.
func init() {
	module.Register(fileHandler{})
}

// ModuleInfo returns the module information.
func (h fileHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: fileModuleID,
		NewInstance: func() module.Module {
			return &fileHandler{
				osOpenFile: fileOsOpenFile,
				osReadFile: fileOsReadFile,
				osClose:    fileOsClose,
				osStat:     fileOsStat,
			}
		},
	}
}

// Check checks the handler configuration.
func (h *fileHandler) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c fileHandlerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.Path == "" {
		report = append(report, fmt.Sprintf("option '%s', missing option or value", "Path"))
	} else {
		f, err := h.osOpenFile(c.Path, os.O_RDONLY, 0)
		if err != nil {
			report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "Path", c.Path))
		} else {
			h.osClose(f)
			fi, err := h.osStat(c.Path)
			if err != nil {
				report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "Path", c.Path))
			}
			if err == nil && fi.IsDir() {
				report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "Path", c.Path))
			}
		}
	}
	if c.StatusCode == nil {
		defaultValue := fileConfigDefaultStatusCode
		c.StatusCode = &defaultValue
	}
	if *c.StatusCode < 100 || *c.StatusCode > 599 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "StatusCode", *c.StatusCode))
	}
	if c.CacheTTL == nil {
		defaultValue := fileConfigDefaultCacheTTL
		c.CacheTTL = &defaultValue
	}
	if *c.CacheTTL < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "CacheTTL", *c.CacheTTL))
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the handler.
func (h *fileHandler) Load(config map[string]interface{}) error {
	var c fileHandlerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	h.config = &c
	h.logger = log.New(os.Stderr, fmt.Sprint(fileLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	if h.config.StatusCode == nil {
		defaultValue := fileConfigDefaultStatusCode
		h.config.StatusCode = &defaultValue
	}
	if h.config.Cache == nil {
		defaultValue := fileConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := fileConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}

	h.rwPool = render.NewRenderWriterPool()
	h.cache = memory.New(time.Duration(*h.config.CacheTTL)*time.Second, 0)

	return nil
}

// Register registers the handler.
func (h *fileHandler) Register(site core.ServerSite) error {
	err := site.RegisterHandler(h)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the handler.
func (h *fileHandler) Start() error {
	err := h.read()
	if err != nil {
		return err
	}

	return nil
}

// Stop stops the handler.
func (h *fileHandler) Stop() {
	h.cache.Clear()
}

// ServeHTTP implements the http handler.
func (h *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if *h.config.Cache {
		obj := h.cache.Get(r.URL.Path)
		if obj != nil {
			render := obj.(render.Render)

			w.WriteHeader(render.StatusCode())
			w.Write(render.Body())

			h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, render.StatusCode(), true)

			return
		}
	}

	err := h.read()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Printf("Render error (url=%s, status=%d)", r.URL.Path, http.StatusServiceUnavailable)

		return
	}

	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	err = h.render(rw, r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Printf("Render error (url=%s, status=%d)", r.URL.Path, http.StatusServiceUnavailable)

		return
	}

	render := rw.Render()

	if *h.config.Cache {
		h.cache.Set(r.URL.Path, render)
	}

	w.WriteHeader(render.StatusCode())
	w.Write(render.Body())

	h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, render.StatusCode(), false)
}

// read reads the file.
func (h *fileHandler) read() error {
	fileInfo, err := h.osStat(h.config.Path)
	if err != nil {
		h.logger.Printf("Failed to stat file '%s': %s", h.config.Path, err)

		return err
	}
	if h.fileInfo == nil || fileInfo.ModTime().After(*h.fileInfo) {
		buf, err := h.osReadFile(h.config.Path)
		if err != nil {
			h.logger.Printf("Failed to read file '%s': %s", h.config.Path, err)

			return err
		}

		h.file = buf
		i := fileInfo.ModTime()
		h.fileInfo = &i
	}

	return nil
}

// render makes a new render.
func (h *fileHandler) render(w render.RenderWriter, r *http.Request) error {
	w.WriteHeader(*h.config.StatusCode)
	w.Write(h.file)

	return nil
}

var _ core.ServerSiteHandlerModule = (*fileHandler)(nil)
