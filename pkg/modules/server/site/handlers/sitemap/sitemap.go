// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// sitemapHandler implements the sitemap handler.
type sitemapHandler struct {
	config               *sitemapHandlerConfig
	logger               *log.Logger
	templateSitemapIndex *template.Template
	templateSitemap      *template.Template
	rwPool               render.RenderWriterPool
	cache                *sitemapHandlerCache
	muCache              *sync.RWMutex
	site                 core.ServerSite
}

// sitemapHandlerConfig implements the sitemap handler configuration.
type sitemapHandlerConfig struct {
	Root         string
	Cache        *bool
	CacheTTL     *int
	Kind         string
	SitemapIndex []SitemapIndexEntry
	Sitemap      []SitemapEntry
}

// SitemapIndexEntry implements a sitemap index entry.
type SitemapIndexEntry struct {
	Name   string
	Type   string
	Static SitemapIndexEntryStatic
}

// SitemapIndexEntryStatic implements a static sitemap index entry.
type SitemapIndexEntryStatic struct {
	Loc string
}

// SitemapEntry implements a sitemap entry.
type SitemapEntry struct {
	Name   string
	Type   string
	Static SitemapEntryStatic
	List   SitemapEntryList
}

// SitemapEntryStatic implements a static sitemap entry.
type SitemapEntryStatic struct {
	Loc        string
	Lastmod    *string
	Changefreq *string
	Priority   *float64
}

// SitemapEntryList implements a sitemap entry list.
type SitemapEntryList struct {
	Resource    string
	Filter      string
	ItemLoc     string
	ItemLastmod *string
	ItemIgnore  *string
	Changefreq  *string
	Priority    *float64
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
	sitemapModuleID module.ModuleID = "server.site.handler.sitemap"

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

// init initializes the module.
func init() {
	module.Register(sitemapHandler{})
}

// ModuleInfo returns the module information.
func (h sitemapHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: sitemapModuleID,
		NewInstance: func() module.Module {
			return &sitemapHandler{
				muCache: new(sync.RWMutex),
			}
		},
	}
}

// Init initializes the handler.
func (h *sitemapHandler) Init(config map[string]interface{}, logger *log.Logger) error {
	h.logger = logger

	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Print("failed to parse configuration")
		return err
	}

	var errInit bool

	if h.config.Root == "" {
		h.logger.Printf("option '%s', missing option or value", "Root")
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
		h.logger.Printf("option '%s', invalid value '%d'", "CacheTTL", *h.config.CacheTTL)
		errInit = true
	}
	var sitemapIndex, sitemap bool
	switch h.config.Kind {
	case "":
		h.logger.Printf("option '%s', missing option or value", "Kind")
		errInit = true
	case sitemapKindSitemapIndex:
		sitemapIndex = true
	case sitemapKindSitemap:
		sitemap = true
	default:
		h.logger.Printf("option '%s', invalid value '%s'", "Kind", h.config.Kind)
		errInit = true
	}

	if sitemapIndex {
		if len(h.config.SitemapIndex) == 0 {
			h.logger.Print("sitemapIndex entry is missing")
			errInit = true
		}
		for _, entry := range h.config.SitemapIndex {
			if entry.Name == "" {
				h.logger.Printf("sitemapIndex entry option '%s', missing option or value", "Name")
				errInit = true
			}
			switch entry.Type {
			case "":
				h.logger.Printf("sitemapIndex entry option '%s', missing option or value", "Type")
				errInit = true
			case sitemapEntrySitemapIndexTypeStatic:
			default:
				h.logger.Printf("sitemapIndex entry option '%s', invalid value '%s'", "Type", entry.Type)
				errInit = true
			}

			if entry.Type == sitemapEntrySitemapIndexTypeStatic {
				if entry.Static.Loc == "" {
					h.logger.Printf("sitemapIndex static entry option '%s', missing option or value", "Loc")
					errInit = true
				}
			}
		}
	}

	if sitemap {
		if len(h.config.Sitemap) == 0 {
			h.logger.Print("sitemap entry is missing")
			errInit = true
		}
		for _, entry := range h.config.Sitemap {
			if entry.Name == "" {
				h.logger.Printf("sitemap entry option '%s', missing option or value", "Name")
			}
			switch entry.Type {
			case "":
				h.logger.Printf("sitemap entry option '%s', missing option or value", "Type")
			case sitemapEntrySitemapTypeStatic:
			case sitemapEntrySitemapTypeList:
			default:
				h.logger.Printf("sitemap entry option '%s', invalid value '%s'", "Type", entry.Type)
			}

			if entry.Type == sitemapEntrySitemapTypeStatic {
				if entry.Static.Loc == "" {
					h.logger.Printf("sitemap static entry option '%s', missing option or value", "Loc")
					errInit = true
				}
				if entry.Static.Lastmod != nil && *entry.Static.Lastmod == "" {
					h.logger.Printf("sitemap static entry option '%s', invalid value '%s'", "Lastmod", *entry.Static.Lastmod)
					errInit = true
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
						h.logger.Printf("sitemap static entry option '%s', invalid value '%s'", "Changefreq",
							*entry.Static.Changefreq)
						errInit = true
					}
				}
				if entry.Static.Priority != nil && (*entry.Static.Priority < 0 || *entry.Static.Priority > 1) {
					h.logger.Printf("sitemap static entry option '%s', invalid value '%.1f'", "Priority", *entry.Static.Priority)
					errInit = true
				}
			}

			if entry.Type == sitemapEntrySitemapTypeList {
				if entry.List.Resource == "" {
					h.logger.Printf("sitemap list entry option '%s', missing option or value", "Resource")
					errInit = true
				}
				if entry.List.Filter == "" {
					h.logger.Printf("sitemap list entry option '%s', missing option or value", "Filter")
					errInit = true
				}
				if entry.List.ItemLoc == "" {
					h.logger.Printf("sitemap list entry option '%s', missing option or value", "ItemLoc")
					errInit = true
				}
				if entry.List.ItemLastmod != nil && *entry.List.ItemLastmod == "" {
					h.logger.Printf("sitemap list entry option '%s', invalid value '%s'", "ItemLastmod",
						*entry.List.ItemLastmod)
					errInit = true
				}
				if entry.List.ItemIgnore != nil && *entry.List.ItemIgnore == "" {
					h.logger.Printf("sitemap list entry option '%s', invalid value '%s'", "ItemIgnore", *entry.List.ItemIgnore)
					errInit = true
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
						h.logger.Printf("sitemap list entry option '%s', invalid value '%s'", "Changefreq", *entry.List.Changefreq)
						errInit = true
					}
				}
				if entry.List.Priority != nil && (*entry.List.Priority < 0 || *entry.List.Priority > 1) {
					h.logger.Printf("sitemap list entry option '%s', invalid value '%.1f'", "Priority", *entry.List.Priority)
					errInit = true
				}
			}
		}
	}

	if errInit {
		return errors.New("init error")
	}

	var err error
	h.templateSitemapIndex, err = template.New("sitemapIndex").Parse(sitemapTemplateSitemapIndex)
	if err != nil {
		return err
	}
	h.templateSitemap, err = template.New("sitemap").Parse(sitemapTemplateSitemap)
	if err != nil {
		return err
	}

	h.rwPool = render.NewRenderWriterPool()

	return nil
}

// Register registers the handler.
func (h *sitemapHandler) Register(site core.ServerSite) error {
	h.site = site

	err := site.RegisterHandler(h)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the handler.
func (h *sitemapHandler) Start() error {
	return nil
}

// Stop stops the handler.
func (h *sitemapHandler) Stop() {
	h.muCache.Lock()
	h.cache = nil
	h.muCache.Unlock()
}

// ServeHTTP implements the http handler.
func (h *sitemapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if *h.config.Cache {
		h.muCache.RLock()
		if h.cache != nil && h.cache.expire.After(time.Now()) {
			render := h.cache.render
			h.muCache.RUnlock()

			w.WriteHeader(render.StatusCode())
			w.Write(render.Body())

			h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, render.StatusCode(), true)

			return
		} else {
			h.muCache.RUnlock()
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
		h.muCache.Lock()
		h.cache = &sitemapHandlerCache{
			render: render,
			expire: time.Now().Add(time.Duration(*h.config.CacheTTL) * time.Second),
		}
		h.muCache.Unlock()
	}

	w.WriteHeader(render.StatusCode())
	w.Write(render.Body())

	h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, render.StatusCode(), false)
}

// render makes a new render.
func (h *sitemapHandler) render(w render.RenderWriter, r *http.Request) error {
	var err error
	switch h.config.Kind {
	case sitemapKindSitemapIndex:
		err = h.sitemapIndex(h.config.SitemapIndex, w, r)
	case sitemapKindSitemap:
		err = h.sitemap(h.config.Sitemap, w, r)
	}
	if err != nil {
		h.logger.Printf("Failed to render: %s", err)

		return err
	}

	w.WriteHeader(http.StatusOK)

	return nil
}

// sitemapIndex writes the sitemap index.
func (h *sitemapHandler) sitemapIndex(s []SitemapIndexEntry, w io.Writer, r *http.Request) error {
	var items []sitemapTemplateSitemapIndexItem
	for _, sitemapEntry := range s {
		items = append(items, sitemapTemplateSitemapIndexItem{
			Loc: h.absURL(sitemapEntry.Static.Loc, h.config.Root),
		})
	}

	err := h.templateSitemapIndex.Execute(w, sitemapTemplateSitemapIndexData{
		Items: items,
	})
	if err != nil {
		return err
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
				return err
			}
			items = append(items, staticItem)

		case sitemapEntrySitemapTypeList:
			listItems, err := h.sitemapTemplateListItems(sitemapEntry)
			if err != nil {
				return err
			}
			items = append(items, listItems...)
		}
	}

	err := h.templateSitemap.Execute(w, sitemapTemplateSitemapData{
		Items: items,
	})
	if err != nil {
		return err
	}

	return nil
}

// sitemapTemplateStaticItem returns a sitemap template static item
func (h *sitemapHandler) sitemapTemplateStaticItem(entry SitemapEntry) (sitemapTemplateSitemapItem, error) {
	item := sitemapTemplateSitemapItem{
		Loc: h.absURL(entry.Static.Loc, h.config.Root),
	}
	if entry.Static.Lastmod != nil {
		item.Lastmod = fmt.Sprintf("%v", *entry.Static.Lastmod)
	}
	if entry.Static.Changefreq != nil {
		item.Changefreq = fmt.Sprintf("%v", *entry.Static.Changefreq)
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
		return nil, err
	}

	for _, data := range resource.Data {
		var jsonData interface{}
		err = json.Unmarshal(data, &jsonData)
		if err != nil {
			return nil, err
		}

		result, err := jsonpath.Get(entry.List.Filter, jsonData)
		if err != nil {
			return nil, err
		}

		elements, ok := result.([]interface{})
		if !ok {
			return nil, nil
		}

		for _, element := range elements {
			var loc, lastmod, ignore string

			itemLoc, err := jsonpath.Get(entry.List.ItemLoc, element)
			if err != nil {
				continue
			}
			if v, ok := itemLoc.(string); ok {
				loc = h.absURL(v, h.config.Root)
			}
			if entry.List.ItemLastmod != nil {
				itemLastmod, err := jsonpath.Get(*entry.List.ItemLastmod, element)
				if err != nil {
					continue
				}
				if v, ok := itemLastmod.(string); ok {
					lastmod = v
				}
			}
			if entry.List.ItemIgnore != nil {
				itemIgnore, err := jsonpath.Get(*entry.List.ItemIgnore, element)
				if err != nil {
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
				item.Changefreq = fmt.Sprintf("%v", *entry.List.Changefreq)
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
