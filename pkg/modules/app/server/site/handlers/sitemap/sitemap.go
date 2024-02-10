package sitemap

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// sitemapHandler implements the sitemap handler.
type sitemapHandler struct {
	config               *sitemapHandlerConfig
	logger               *slog.Logger
	templateSitemapIndex *template.Template
	templateSitemap      *template.Template
	rwPool               render.RenderWriterPool
	cache                *sitemapHandlerCache
	muCache              *sync.RWMutex
	site                 core.ServerSite
}

// sitemapHandlerConfig implements the sitemap handler configuration.
type sitemapHandlerConfig struct {
	Root         string              `mapstructure:"root"`
	Cache        *bool               `mapstructure:"cache"`
	CacheTTL     *int                `mapstructure:"cacheTTL"`
	Kind         string              `mapstructure:"kind"`
	SitemapIndex []SitemapIndexEntry `mapstructure:"sitemapIndex"`
	Sitemap      []SitemapEntry      `mapstructure:"sitemap"`
}

// SitemapIndexEntry implements a sitemap index entry.
type SitemapIndexEntry struct {
	Name   string                  `mapstructure:"name"`
	Type   string                  `mapstructure:"type"`
	Static SitemapIndexEntryStatic `mapstructure:"static"`
}

// SitemapIndexEntryStatic implements a static sitemap index entry.
type SitemapIndexEntryStatic struct {
	Loc string `mapstructure:"loc"`
}

// SitemapEntry implements a sitemap entry.
type SitemapEntry struct {
	Name   string             `mapstructure:"name"`
	Type   string             `mapstructure:"type"`
	Static SitemapEntryStatic `mapstructure:"static"`
	List   SitemapEntryList   `mapstructure:"list"`
}

// SitemapEntryStatic implements a static sitemap entry.
type SitemapEntryStatic struct {
	Loc        string   `mapstructure:"loc"`
	Lastmod    *string  `mapstructure:"lastmod"`
	Changefreq *string  `mapstructure:"changefreq"`
	Priority   *float64 `mapstructure:"priority"`
}

// SitemapEntryList implements a sitemap entry list.
type SitemapEntryList struct {
	Resource    string   `mapstructure:"resource"`
	Filter      string   `mapstructure:"filter"`
	ItemLoc     string   `mapstructure:"itemLoc"`
	ItemLastmod *string  `mapstructure:"itemLastmod"`
	ItemIgnore  *string  `mapstructure:"itemIgnore"`
	Changefreq  *string  `mapstructure:"changefreq"`
	Priority    *float64 `mapstructure:"priority"`
}

// sitemapTemplateSitemapIndexData implements the sitemap index template data.
type sitemapTemplateSitemapIndexData struct {
	Items []sitemapTemplateSitemapIndexItem
}

// sitemapTemplateSitemapIndexItem implements a sitemap index template item.
type sitemapTemplateSitemapIndexItem struct {
	Loc string
}

// sitemapTemplateSitemapData implements the sitemap template data.
type sitemapTemplateSitemapData struct {
	Items []sitemapTemplateSitemapItem
}

// sitemapTemplateSitemapIndexEntry implements a sitemap template item.
type sitemapTemplateSitemapItem struct {
	Loc        string
	Lastmod    string
	Changefreq string
	Priority   string
}

// sitemapHandlerCache implements the sitemap handler cache.
type sitemapHandlerCache struct {
	render render.Render
	expire time.Time
}

const (
	sitemapModuleID module.ModuleID = "app.server.site.handler.sitemap"

	sitemapKindSitemapIndex            string = "sitemapIndex"
	sitemapKindSitemap                 string = "sitemap"
	sitemapEntrySitemapIndexTypeStatic string = "static"
	sitemapEntrySitemapTypeStatic      string = "static"
	sitemapEntrySitemapTypeList        string = "list"
	sitemapChangefreqAlways            string = "always"
	sitemapChangefreqHourly            string = "hourly"
	sitemapChangefreqDaily             string = "daily"
	sitemapChangefreqWeekly            string = "weekly"
	sitemapChangefreqMonthly           string = "monthly"
	sitemapChangefreqYearly            string = "yearly"
	sitemapChangefreqNever             string = "never"

	sitemapConfigDefaultCache    bool = false
	sitemapConfigDefaultCacheTTL int  = 60
)

var (
	//go:embed templates/sitemapindex.xml.tmpl
	sitemapTemplateSitemapIndex string
	//go:embed templates/sitemap.xml.tmpl
	sitemapTemplateSitemap string
)

// init initializes the package.
func init() {
	module.Register(sitemapHandler{})
}

// ModuleInfo returns the module information.
func (h sitemapHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: sitemapModuleID,
		NewInstance: func() module.Module {
			return &sitemapHandler{
				logger:  slog.New(log.NewHandler(os.Stderr, string(sitemapModuleID), nil)),
				muCache: new(sync.RWMutex),
			}
		},
	}
}

// Init initializes the handler.
func (h *sitemapHandler) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if h.config.Root == "" {
		h.logger.Error("Missing option or value", "option", "Root")
		errConfig = true
	}
	if h.config.Cache == nil {
		defaultValue := sitemapConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := sitemapConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}
	if *h.config.CacheTTL < 0 {
		h.logger.Error("Invalid value", "option", "CacheTTL", "value", *h.config.CacheTTL)
		errConfig = true
	}
	var sitemapIndex, sitemap bool
	switch h.config.Kind {
	case "":
		h.logger.Error("Missing option or value", "option", "Kind")
		errConfig = true
	case sitemapKindSitemapIndex:
		sitemapIndex = true
	case sitemapKindSitemap:
		sitemap = true
	default:
		h.logger.Error("Invalid value", "option", "Kind", "value", h.config.Kind)
		errConfig = true
	}

	if sitemapIndex {
		if len(h.config.SitemapIndex) == 0 {
			h.logger.Error("Entry is missing", "kind", "sitemapIndex")
			errConfig = true
		}
		for index, entry := range h.config.SitemapIndex {
			if entry.Name == "" {
				h.logger.Error("Missing option or value", "kind", "sitemapIndex", "entry", index+1, "option", "Name")
				errConfig = true
			}
			switch entry.Type {
			case "":
				h.logger.Error("Missing option or value", "kind", "sitemapIndex", "entry", index+1, "option", "Type")
				errConfig = true
			case sitemapEntrySitemapIndexTypeStatic:
			default:
				h.logger.Error("Invalid value", "kind", "sitemapIndex", "entry", index+1, "option", "Type", "value", entry.Type)
				errConfig = true
			}

			if entry.Type == sitemapEntrySitemapIndexTypeStatic {
				if entry.Static.Loc == "" {
					h.logger.Error("Missing option or value", "kind", "sitemapIndex", "entry", index+1, "type", "static",
						"option", "Loc")
					errConfig = true
				}
			}
		}
	}

	if sitemap {
		if len(h.config.Sitemap) == 0 {
			h.logger.Error("Entry is missing", "kind", "sitemap")
			errConfig = true
		}
		for index, entry := range h.config.Sitemap {
			if entry.Name == "" {
				h.logger.Error("Missing option or value", "kind", "sitemap", "entry", index+1, "option", "Name")
			}
			switch entry.Type {
			case "":
				h.logger.Error("Missing option or value", "kind", "sitemap", "entry", index+1, "option", "Type")
			case sitemapEntrySitemapTypeStatic:
			case sitemapEntrySitemapTypeList:
			default:
				h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "option", "Type", "value", entry.Type)
			}

			if entry.Type == sitemapEntrySitemapTypeStatic {
				if entry.Static.Loc == "" {
					h.logger.Error("Missing option or value", "kind", "sitemap", "entry", index+1, "type", "static",
						"option", "Loc")
					errConfig = true
				}
				if entry.Static.Lastmod != nil && *entry.Static.Lastmod == "" {
					h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "type", "static", "option", "Lastmod",
						"value", *entry.Static.Lastmod)
					errConfig = true
				}
				if entry.Static.Changefreq != nil {
					switch *entry.Static.Changefreq {
					case sitemapChangefreqAlways:
					case sitemapChangefreqHourly:
					case sitemapChangefreqDaily:
					case sitemapChangefreqWeekly:
					case sitemapChangefreqMonthly:
					case sitemapChangefreqYearly:
					case sitemapChangefreqNever:
					default:
						h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "type", "static",
							"option", "Changefreq", "value", *entry.Static.Changefreq)
						errConfig = true
					}
				}
				if entry.Static.Priority != nil && (*entry.Static.Priority < 0 || *entry.Static.Priority > 1) {
					h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "type", "static", "option", "Priority",
						"value", *entry.Static.Priority)
					errConfig = true
				}
			}

			if entry.Type == sitemapEntrySitemapTypeList {
				if entry.List.Resource == "" {
					h.logger.Error("Missing option or value", "kind", "sitemap", "entry", index+1, "type", "list",
						"option", "Resource")
					errConfig = true
				}
				if entry.List.Filter == "" {
					h.logger.Error("Missing option or value", "kind", "sitemap", "entry", index+1, "type", "list",
						"option", "Filter")
					errConfig = true
				}
				if entry.List.ItemLoc == "" {
					h.logger.Error("Missing option or value", "kind", "sitemap", "entry", index+1, "type", "list",
						"option", "ItemLoc")
					errConfig = true
				}
				if entry.List.ItemLastmod != nil && *entry.List.ItemLastmod == "" {
					h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "option", "type", "list",
						"ItemLastmod", "value", *entry.List.ItemLastmod)
					errConfig = true
				}
				if entry.List.ItemIgnore != nil && *entry.List.ItemIgnore == "" {
					h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "option", "type", "list",
						"ItemIgnore", "value", *entry.List.ItemIgnore)
					errConfig = true
				}
				if entry.List.Changefreq != nil {
					switch *entry.List.Changefreq {
					case sitemapChangefreqAlways:
					case sitemapChangefreqHourly:
					case sitemapChangefreqDaily:
					case sitemapChangefreqWeekly:
					case sitemapChangefreqMonthly:
					case sitemapChangefreqYearly:
					case sitemapChangefreqNever:
					default:
						h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "type", "list", "option", "Changefreq",
							"value", *entry.List.Changefreq)
						errConfig = true
					}
				}
				if entry.List.Priority != nil && (*entry.List.Priority < 0 || *entry.List.Priority > 1) {
					h.logger.Error("Invalid value", "kind", "sitemap", "entry", index+1, "type", "list", "option", "Priority",
						"value", *entry.List.Priority)
					errConfig = true
				}
			}
		}
	}

	if errConfig {
		return errors.New("config")
	}

	var err error
	if h.config.Kind == sitemapKindSitemapIndex {
		h.templateSitemapIndex, err = template.New("sitemapIndex").Parse(sitemapTemplateSitemapIndex)
	}
	if h.config.Kind == sitemapKindSitemap {
		h.templateSitemap, err = template.New("sitemap").Parse(sitemapTemplateSitemap)
	}
	if err != nil {
		return fmt.Errorf("parse template: %v", err)
	}

	h.rwPool = render.NewRenderWriterPool()

	return nil
}

// Register registers the handler.
func (h *sitemapHandler) Register(site core.ServerSite) error {
	h.site = site

	if err := site.RegisterHandler(h); err != nil {
		return fmt.Errorf("register handler: %v", err)
	}

	return nil
}

// Start starts the handler.
func (h *sitemapHandler) Start() error {
	return nil
}

// Stop stops the handler.
func (h *sitemapHandler) Stop() error {
	h.muCache.Lock()
	h.cache = nil
	h.muCache.Unlock()

	return nil
}

// ServeHTTP implements the http handler.
func (h *sitemapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

			h.logger.Info("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", true)

			return
		} else {
			h.muCache.RUnlock()
		}
	}

	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	if err := h.render(rw, r); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

		return
	}

	render := rw.Render()

	if *h.config.Cache {
		h.muCache.Lock()
		h.cache = &sitemapHandlerCache{
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

	h.logger.Info("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", false)
}

// render makes a new render.
func (h *sitemapHandler) render(w render.RenderWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)

	var err error
	switch h.config.Kind {
	case sitemapKindSitemapIndex:
		err = h.sitemapIndex(h.config.SitemapIndex, w, r)
	case sitemapKindSitemap:
		err = h.sitemap(h.config.Sitemap, w, r)
	}
	if err != nil {
		h.logger.Error("Failed to generate sitemap", "err", err)
		return fmt.Errorf("generate sitemap: %v", err)
	}

	return nil
}

// sitemapIndex writes the sitemap index.
func (h *sitemapHandler) sitemapIndex(s []SitemapIndexEntry, w io.Writer, r *http.Request) error {
	items := make([]sitemapTemplateSitemapIndexItem, 0, len(s))
	for _, sitemapEntry := range s {
		items = append(items, sitemapTemplateSitemapIndexItem{
			Loc: h.absURL(sitemapEntry.Static.Loc, h.config.Root),
		})
	}

	if err := h.templateSitemapIndex.Execute(w, sitemapTemplateSitemapIndexData{
		Items: items,
	}); err != nil {
		return fmt.Errorf("execute template: %v", err)
	}

	return nil
}

// sitemap writes the sitemap.
func (h *sitemapHandler) sitemap(s []SitemapEntry, w io.Writer, r *http.Request) error {
	var items []sitemapTemplateSitemapItem
	for _, sitemapEntry := range s {
		switch sitemapEntry.Type {
		case sitemapEntrySitemapTypeStatic:
			staticItem, err := h.sitemapTemplateStaticItem(sitemapEntry)
			if err != nil {
				return fmt.Errorf("write static entry: %v", err)
			}
			items = append(items, staticItem)

		case sitemapEntrySitemapTypeList:
			listItems, err := h.sitemapTemplateListItems(sitemapEntry)
			if err != nil {
				return fmt.Errorf("write list entry: %v", err)
			}
			items = append(items, listItems...)
		}
	}

	if err := h.templateSitemap.Execute(w, sitemapTemplateSitemapData{
		Items: items,
	}); err != nil {
		return fmt.Errorf("execute template: %v", err)
	}

	return nil
}

// sitemapTemplateStaticItem returns a sitemap template static item
func (h *sitemapHandler) sitemapTemplateStaticItem(entry SitemapEntry) (sitemapTemplateSitemapItem, error) {
	item := sitemapTemplateSitemapItem{
		Loc: h.absURL(entry.Static.Loc, h.config.Root),
	}
	if entry.Static.Lastmod != nil {
		item.Lastmod = *entry.Static.Lastmod
	}
	if entry.Static.Changefreq != nil {
		item.Changefreq = *entry.Static.Changefreq
	}
	if entry.Static.Priority != nil {
		item.Priority = fmt.Sprintf("%v", *entry.Static.Priority)
	}

	return item, nil
}

// sitemapTemplateListItems returns the sitemap template list items
func (h *sitemapHandler) sitemapTemplateListItems(entry SitemapEntry) ([]sitemapTemplateSitemapItem, error) {
	var items []sitemapTemplateSitemapItem

	resource, err := h.site.Store().LoadResource(entry.List.Resource)
	if err != nil {
		return nil, fmt.Errorf("load resource %s: %v", entry.List.Resource, err)
	}

	for _, data := range resource.Data {
		var jsonData interface{}
		if err := json.Unmarshal(data, &jsonData); err != nil {
			return nil, fmt.Errorf("parse resource %s data: %v", entry.List.Resource, err)
		}

		result, err := jsonpath.Get(entry.List.Filter, jsonData)
		if err != nil {
			return nil, fmt.Errorf("filter resource %s data: %v", entry.List.Resource, err)
		}

		elements, ok := result.([]interface{})
		if !ok {
			return nil, fmt.Errorf("parse resource %s result: %v", entry.List.Resource, err)
		}

		for _, element := range elements {
			var loc, lastmod, ignore string

			itemLoc, err := jsonpath.Get(entry.List.ItemLoc, element)
			if err != nil {
				h.logger.Debug("Failed to extract loc from resource item", "resource", entry.List.Resource)
				continue
			}
			if v, ok := itemLoc.(string); ok {
				loc = h.absURL(v, h.config.Root)
			}
			if entry.List.ItemLastmod != nil {
				itemLastmod, err := jsonpath.Get(*entry.List.ItemLastmod, element)
				if err != nil {
					h.logger.Debug("Failed to extract lastmod from resource item", "resource", entry.List.Resource)
					continue
				}
				if v, ok := itemLastmod.(string); ok {
					lastmod = v
				}
			}
			if entry.List.ItemIgnore != nil {
				itemIgnore, err := jsonpath.Get(*entry.List.ItemIgnore, element)
				if err != nil {
					h.logger.Debug("Failed to extract ignore from resource item", "resource", entry.List.Resource)
					continue
				}
				switch v := itemIgnore.(type) {
				case string:
					ignore = v
				case bool:
					ignore = strconv.FormatBool(v)
				case int64:
					ignore = strconv.FormatInt(v, 10)
				}
				if strings.EqualFold(ignore, "true") || strings.EqualFold(ignore, "1") {
					continue
				}
			}

			item := sitemapTemplateSitemapItem{
				Loc:     loc,
				Lastmod: lastmod,
			}
			if entry.List.Changefreq != nil {
				item.Changefreq = *entry.List.Changefreq
			}
			if entry.List.Priority != nil {
				item.Priority = fmt.Sprintf("%v", *entry.List.Priority)
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// absURL returns the absolute form of the given URL.
func (h *sitemapHandler) absURL(url string, root string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return fmt.Sprintf("%s%s", root, url)
}

var _ core.ServerSiteHandlerModule = (*sitemapHandler)(nil)
