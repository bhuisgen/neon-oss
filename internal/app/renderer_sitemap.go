// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// sitemapRenderer implements the sitemap renderer
type sitemapRenderer struct {
	config               *SitemapRendererConfig
	logger               *log.Logger
	templateSitemapIndex *template.Template
	templateSitemap      *template.Template
	bufferPool           BufferPool
	cache                Cache
	fetcher              Fetcher
	next                 Renderer
}

// SitemapRendererConfig implements the sitemap renderer configuration
type SitemapRendererConfig struct {
	Root     string
	Cache    bool
	CacheTTL int
	Routes   []SitemapRoute
}

// SitemapRoute implements a sitemap route
type SitemapRoute struct {
	Path         string
	Kind         string
	SitemapIndex []SitemapIndexEntry
	Sitemap      []SitemapEntry
}

// SitemapIndexEntry implements a sitemap index entry
type SitemapIndexEntry struct {
	Name   string
	Type   string
	Static SitemapIndexEntryStatic
}

// SitemapIndexEntryStatic implements a static sitemap index entry
type SitemapIndexEntryStatic struct {
	Loc string
}

// SitemapEntry implements a sitemap entry
type SitemapEntry struct {
	Name   string
	Type   string
	Static SitemapEntryStatic
	List   SitemapEntryList
}

// SitemapEntryStatic implements a static sitemap entry
type SitemapEntryStatic struct {
	Loc        string
	Lastmod    *string
	Changefreq *string
	Priority   *float32
}

// SitemapEntryList implements a sitemap entry list
type SitemapEntryList struct {
	Resource                   string
	ResourcePayloadItems       string
	ResourcePayloadItemLoc     string
	ResourcePayloadItemLastmod *string
	ResourcePayloadItemIgnore  *string
	Changefreq                 *string
	Priority                   *float32
}

// sitemapRender implements a render
type sitemapRender struct {
	Body   []byte
	Status int
}

// sitemapTemplateSitemapIndexData implements the sitemap index template data
type sitemapTemplateSitemapIndexData struct {
	Items []sitemapTemplateSitemapIndexItem
}

// sitemapTemplateSitemapIndexItem implements a sitemap index template item
type sitemapTemplateSitemapIndexItem struct {
	Loc string
}

// sitemapTemplateSitemapData implements the sitemap template data
type sitemapTemplateSitemapData struct {
	Items []sitemapTemplateSitemapItem
}

// sitemapTemplateSitemapIndexEntry implements a sitemap template item
type sitemapTemplateSitemapItem struct {
	Loc        string
	Lastmod    string
	Changefreq string
	Priority   string
}

const (
	sitemapLogger                      string = "server[sitemap]"
	sitemapKindSitemapIndex            string = "sitemap_index"
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
)

var (
	//go:embed templates/sitemap/sitemapindex.xml.tmpl
	sitemapTemplateSitemapIndex string
	//go:embed templates/sitemap/sitemap.xml.tmpl
	sitemapTemplateSitemap string
)

// CreateSitemapRenderer creates a new sitemap renderer
func CreateSitemapRenderer(config *SitemapRendererConfig, fetcher Fetcher) (*sitemapRenderer, error) {
	templateSitemapIndex, err := template.New("sitemap_index").Parse(sitemapTemplateSitemapIndex)
	if err != nil {
		return nil, err
	}

	templateSitemap, err := template.New("sitemap").Parse(sitemapTemplateSitemap)
	if err != nil {
		return nil, err
	}

	return &sitemapRenderer{
		config:               config,
		logger:               log.New(os.Stderr, fmt.Sprint(sitemapLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		templateSitemapIndex: templateSitemapIndex,
		templateSitemap:      templateSitemap,
		bufferPool:           newBufferPool(),
		cache:                newCache(),
		fetcher:              fetcher,
	}, nil
}

// Handle implements the renderer
func (r *sitemapRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
	var routeIndex int = -1
	for index, route := range r.config.Routes {
		if route.Path != req.URL.Path {
			continue
		}

		routeIndex = index

		break
	}
	if routeIndex == -1 {
		r.next.Handle(w, req, i)

		return
	}

	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*sitemapRender)
			w.WriteHeader(result.Status)
			w.Write(result.Body)

			r.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", req.URL.Path, result.Status, true)

			return
		}
	}

	b := r.bufferPool.Get()
	defer r.bufferPool.Put(b)

	err := r.render(routeIndex, req, b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, http.StatusInternalServerError)

		return
	}

	if r.config.Cache {
		body := make([]byte, b.Len())
		copy(body, b.Bytes())

		r.cache.Set(req.URL.Path, &sitemapRender{
			Body:   body,
			Status: http.StatusOK,
		}, time.Duration(r.config.CacheTTL)*time.Second)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b.Bytes())

	r.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", req.URL.Path, http.StatusOK, false)
}

// Next configures the next renderer
func (r *sitemapRenderer) Next(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *sitemapRenderer) render(routeIndex int, req *http.Request, w io.Writer) error {
	var err error
	switch r.config.Routes[routeIndex].Kind {
	case sitemapKindSitemapIndex:
		err = r.sitemapIndex(r.config.Routes[routeIndex].SitemapIndex, req, w)
	case sitemapKindSitemap:
		err = r.sitemap(r.config.Routes[routeIndex].Sitemap, req, w)
	}
	if err != nil {
		r.logger.Printf("Failed to render: %s", err)

		return err
	}

	return nil
}

// sitemapIndex generates a sitemap index
func (r *sitemapRenderer) sitemapIndex(s []SitemapIndexEntry, req *http.Request, w io.Writer) error {
	var items []sitemapTemplateSitemapIndexItem
	for _, sitemapEntry := range s {
		items = append(items, sitemapTemplateSitemapIndexItem{
			Loc: r.absURL(sitemapEntry.Static.Loc, r.config.Root),
		})
	}

	err := r.templateSitemapIndex.Execute(w, sitemapTemplateSitemapIndexData{
		Items: items,
	})
	if err != nil {
		return err
	}

	return nil
}

// sitemap generates a sitemap
func (r *sitemapRenderer) sitemap(s []SitemapEntry, req *http.Request, w io.Writer) error {
	var items []sitemapTemplateSitemapItem
	for _, sitemapEntry := range s {
		switch sitemapEntry.Type {
		case sitemapEntrySitemapTypeStatic:
			item := sitemapTemplateSitemapItem{
				Loc: r.absURL(sitemapEntry.Static.Loc, r.config.Root),
			}
			if sitemapEntry.Static.Lastmod != nil {
				item.Lastmod = fmt.Sprintf("%v", *sitemapEntry.Static.Lastmod)
			}
			if sitemapEntry.Static.Changefreq != nil {
				item.Changefreq = fmt.Sprintf("%v", *sitemapEntry.Static.Changefreq)
			}
			if sitemapEntry.Static.Priority != nil {
				item.Priority = fmt.Sprintf("%v", *sitemapEntry.Static.Priority)
			}
			items = append(items, item)

		case sitemapEntrySitemapTypeList:
			response, err := r.fetcher.Get(sitemapEntry.List.Resource)
			if err != nil {
				continue
			}

			var payload interface{}
			err = json.Unmarshal(response, &payload)
			if err != nil {
				continue
			}

			mPayload := payload.(map[string]interface{})
			responseData := mPayload[sitemapEntry.List.ResourcePayloadItems]
			payloadDataArray := responseData.([]interface{})

			for _, item := range payloadDataArray {
				mItem := item.(map[string]interface{})

				var loc, lastmod, ignore string
				if v, ok := mItem[sitemapEntry.List.ResourcePayloadItemLoc].(string); ok {
					loc = r.absURL(v, r.config.Root)
				} else {
					continue
				}
				if sitemapEntry.List.ResourcePayloadItemLastmod != nil {
					if v, ok := mItem[*sitemapEntry.List.ResourcePayloadItemLastmod].(string); ok {
						lastmod = v
					}
				}
				if sitemapEntry.List.ResourcePayloadItemIgnore != nil {
					switch value := mItem[*sitemapEntry.List.ResourcePayloadItemIgnore].(type) {
					case string:
						ignore = value
					case bool:
						ignore = strconv.FormatBool(value)
					case int64:
						ignore = strconv.FormatInt(value, 10)
					}
				}
				if strings.EqualFold(ignore, "true") || strings.EqualFold(ignore, "1") {
					continue
				}

				entry := sitemapTemplateSitemapItem{
					Loc:     loc,
					Lastmod: lastmod,
				}
				if sitemapEntry.List.Changefreq != nil {
					entry.Changefreq = fmt.Sprintf("%v", *sitemapEntry.List.Changefreq)
				}
				if sitemapEntry.List.Priority != nil {
					entry.Priority = fmt.Sprintf("%v", *sitemapEntry.List.Priority)
				}
				items = append(items, entry)
			}
		}
	}

	err := r.templateSitemap.Execute(w, sitemapTemplateSitemapData{
		Items: items,
	})
	if err != nil {
		return err
	}

	return nil
}

// absURL returns the absolute form of the given URL
func (r *sitemapRenderer) absURL(url string, root string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return fmt.Sprintf("%s%s", root, url)
}
