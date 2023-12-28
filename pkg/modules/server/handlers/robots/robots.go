// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Unauthorized copying of this file, via any medium is strictly prohibited.

package robots

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/cache"
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
	cache    cache.Cache
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

const (
	robotsModuleID module.ModuleID = "server.handler.robots"
	robotsLogger   string          = "server.handler.robots"

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

// Check checks the handler configuration.
func (h *robotsHandler) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c robotsHandlerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	for _, item := range c.Hosts {
		if item == "" {
			report = append(report, fmt.Sprintf("option '%s', missing option or value", "Hosts"))
		}
	}
	if c.CacheTTL == nil {
		defaultValue := robotsConfigDefaultCacheTTL
		c.CacheTTL = &defaultValue
	}
	if *c.CacheTTL < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "CacheTTL", *c.CacheTTL))
	}
	for _, item := range c.Sitemaps {
		if item == "" {
			report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "Sitemaps", item))
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the handler.
func (h *robotsHandler) Load(config map[string]interface{}) error {
	var c robotsHandlerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	h.config = &c
	h.logger = log.New(os.Stderr, fmt.Sprint(robotsLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	if h.config.Cache == nil {
		defaultValue := robotsConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := robotsConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}

	h.template, err = template.New("robots").Parse(robotsTemplate)
	if err != nil {
		return err
	}
	h.rwPool = render.NewRenderWriterPool()
	h.cache = cache.NewCache()

	return nil
}

// Register registers the server resources.
func (h *robotsHandler) Register(registry core.ServerRegistry) error {
	err := registry.RegisterHandler(h)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the handler.
func (h *robotsHandler) Start(store core.Store, fetcher core.Fetcher) error {
	return nil
}

// Mount mounts the handler.
func (h *robotsHandler) Mount() error {
	return nil
}

// Unmount unmounts the handler.
func (h *robotsHandler) Unmount() {
}

// Stop stops the handler.
func (h *robotsHandler) Stop() {
	h.cache.Clear()
}

// ServeHTTP implements the http handler.
func (h *robotsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		h.cache.Set(r.URL.Path, render, time.Duration(*h.config.CacheTTL)*time.Second)
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

var _ core.ServerHandlerModule = (*robotsHandler)(nil)
