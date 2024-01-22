// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package robots

import (
	_ "embed"
	"errors"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// robotsHandler implements the robots handler.
type robotsHandler struct {
	config   *robotsHandlerConfig
	logger   *log.Logger
	template *template.Template
	rwPool   render.RenderWriterPool
	cache    *robotsHandlerCache
}

// robotsHandlerConfig implements the robots handler configuration.
type robotsHandlerConfig struct {
	Hosts    []string
	Cache    *bool
	CacheTTL *int
	Sitemaps []string
}

// robotsTemplateData implements the robots template data.
type robotsTemplateData struct {
	Check    bool
	Sitemaps []string
}

// robotsHandlerCache implements the robots handler cache.
type robotsHandlerCache struct {
	render render.Render
	expire time.Time
}

const (
	robotsModuleID module.ModuleID = "server.site.handler.robots"

	robotsConfigDefaultCache    bool = false
	robotsConfigDefaultCacheTTL int  = 60
)

var (
	//go:embed templates/robots.txt.tmpl
	robotsTemplate string
)

// init initializes the module.
func init() {
	module.Register(robotsHandler{})
}

// ModuleInfo returns the module information.
func (h robotsHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: robotsModuleID,
		NewInstance: func() module.Module {
			return &robotsHandler{}
		},
	}
}

// Init initializes the handler.
func (h *robotsHandler) Init(config map[string]interface{}, logger *log.Logger) error {
	h.logger = logger

	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Print("failed to parse configuration")
		return err
	}

	var errInit bool

	for _, item := range h.config.Hosts {
		if item == "" {
			h.logger.Printf("option '%s', missing option or value", "Hosts")
			errInit = true
		}
	}
	if h.config.Cache == nil {
		defaultValue := robotsConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := robotsConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}
	if *h.config.CacheTTL < 0 {
		h.logger.Printf("option '%s', invalid value '%d'", "CacheTTL", *h.config.CacheTTL)
		errInit = true
	}
	for _, item := range h.config.Sitemaps {
		if item == "" {
			h.logger.Printf("option '%s', invalid value '%s'", "Sitemaps", item)
			errInit = true
		}
	}

	if errInit {
		return errors.New("init error")
	}

	var err error
	h.template, err = template.New("robots").Parse(robotsTemplate)
	if err != nil {
		return err
	}

	h.rwPool = render.NewRenderWriterPool()

	return nil
}

// Register registers the handler.
func (h *robotsHandler) Register(site core.ServerSite) error {
	err := site.RegisterHandler(h)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the handler.
func (h *robotsHandler) Start() error {
	return nil
}

// Stop stops the handler.
func (h *robotsHandler) Stop() {
	h.cache = nil
}

// ServeHTTP implements the http handler.
func (h *robotsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if *h.config.Cache {
		if h.cache != nil && h.cache.expire.After(time.Now()) {
			render := h.cache.render

			w.WriteHeader(render.StatusCode())
			w.Write(render.Body())

			h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, render.StatusCode(), true)

			return
		}
	}

	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	err := h.render(rw, r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Printf("Render error (url=%s, status=%d)", r.URL.Path, http.StatusServiceUnavailable)

		return
	}

	render := rw.Render()

	if *h.config.Cache {
		h.cache = &robotsHandlerCache{
			render: render,
			expire: time.Now().Add(time.Duration(*h.config.CacheTTL) * time.Second),
		}
	}

	w.WriteHeader(render.StatusCode())
	w.Write(render.Body())

	h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, render.StatusCode(), false)
}

// render makes a new render.
func (h *robotsHandler) render(w render.RenderWriter, r *http.Request) error {
	var check bool
	for _, host := range h.config.Hosts {
		if host == r.Host {
			check = true
		}
	}

	err := h.template.Execute(w, robotsTemplateData{
		Check:    check,
		Sitemaps: h.config.Sitemaps,
	})
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)

	return nil
}

var _ core.ServerSiteHandlerModule = (*robotsHandler)(nil)
