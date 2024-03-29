package robots

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// robotsHandler implements the robots handler.
type robotsHandler struct {
	config   *robotsHandlerConfig
	logger   *slog.Logger
	template *template.Template
	rwPool   render.RenderWriterPool
	cache    *robotsHandlerCache
	muCache  *sync.RWMutex
}

// robotsHandlerConfig implements the robots handler configuration.
type robotsHandlerConfig struct {
	Hosts    []string `mapstructure:"hosts"`
	Cache    *bool    `mapstructure:"cache"`
	CacheTTL *int     `mapstructure:"cacheTTL"`
	Sitemaps []string `mapstructure:"sitemaps"`
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
	robotsModuleID module.ModuleID = "app.server.site.handler.robots"

	robotsConfigDefaultCache    bool = false
	robotsConfigDefaultCacheTTL int  = 60
)

var (
	//go:embed templates/robots.txt.tmpl
	robotsTemplate string
)

// init initializes the package.
func init() {
	module.Register(robotsHandler{})
}

// ModuleInfo returns the module information.
func (h robotsHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           robotsModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &robotsHandler{
				logger:  slog.New(log.NewHandler(os.Stderr, string(robotsModuleID), nil)),
				muCache: new(sync.RWMutex),
			}
		},
	}
}

// Init initializes the handler.
func (h *robotsHandler) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	for _, item := range h.config.Hosts {
		if item == "" {
			h.logger.Error("Missing option or value", "option", "Hosts")
			errConfig = true
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
	if *h.config.CacheTTL <= 0 {
		h.logger.Error("Invalid value", "option", "CacheTTL", "value", *h.config.CacheTTL)
		errConfig = true
	}
	for _, item := range h.config.Sitemaps {
		if item == "" {
			h.logger.Error("Invalid value", "option", "Sitemaps", "value", item)
			errConfig = true
		}
	}

	if errConfig {
		return errors.New("config")
	}

	var err error
	h.template, err = template.New("robots").Parse(robotsTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %v", err)
	}

	h.rwPool = render.NewRenderWriterPool()

	return nil
}

// Register registers the handler.
func (h *robotsHandler) Register(site core.ServerSite) error {
	if err := site.RegisterHandler(h); err != nil {
		return fmt.Errorf("register handler: %v", err)
	}

	return nil
}

// Start starts the handler.
func (h *robotsHandler) Start() error {
	return nil
}

// Stop stops the handler.
func (h *robotsHandler) Stop() error {
	h.muCache.Lock()
	h.cache = nil
	h.muCache.Unlock()

	return nil
}

// ServeHTTP implements the http handler.
func (h *robotsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if *h.config.Cache {
		h.muCache.RLock()
		if h.cache != nil && h.cache.expire.After(time.Now()) {
			render := h.cache.render
			h.muCache.RUnlock()

			w.WriteHeader(render.StatusCode())
			if _, err := w.Write(render.Body()); err != nil {
				h.logger.Error("Failed to write render", "err", err)
				return
			}

			h.logger.Debug("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", true)

			return
		} else {
			h.muCache.RUnlock()
		}
	}

	render, err := h.render(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

		return
	}

	if *h.config.Cache {
		h.muCache.Lock()
		h.cache = &robotsHandlerCache{
			render: render,
			expire: time.Now().Add(time.Duration(*h.config.CacheTTL) * time.Second),
		}
		h.muCache.Unlock()
	}

	w.WriteHeader(render.StatusCode())
	if _, err := w.Write(render.Body()); err != nil {
		h.logger.Error("Failed to write render", "err", err)
		return
	}

	h.logger.Info("Render completed ", "url", r.URL.Path, "status", render.StatusCode(), "cache", false)
}

// render makes a new render.
func (h *robotsHandler) render(r *http.Request) (render.Render, error) {
	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)

	var check bool
	for _, host := range h.config.Hosts {
		if host == r.Host {
			check = true
		}
	}

	if err := h.template.Execute(rw, robotsTemplateData{
		Check:    check,
		Sitemaps: h.config.Sitemaps,
	}); err != nil {
		h.logger.Error("Failed to execute template", "err", err)
		return nil, fmt.Errorf("execute template: %v", err)
	}

	return rw.Render(), nil
}

var _ core.ServerSiteHandlerModule = (*robotsHandler)(nil)
