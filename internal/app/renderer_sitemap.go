// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// sitemapRenderer implements the sitemap renderer
type sitemapRenderer struct {
	config     *SitemapRendererConfig
	logger     *log.Logger
	bufferPool BufferPool
	cache      Cache
	fetcher    Fetcher
	next       Renderer
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
	Changefreq                 *string
	Priority                   *float32
}

// sitemapRender implements a render
type sitemapRender struct {
	Body   []byte
	Status int
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

// CreateSitemapRenderer creates a new sitemap renderer
func CreateSitemapRenderer(config *SitemapRendererConfig, fetcher Fetcher) (*sitemapRenderer, error) {
	return &sitemapRenderer{
		config:     config,
		logger:     log.New(os.Stderr, fmt.Sprint(sitemapLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		bufferPool: newBufferPool(),
		cache:      newCache(),
		fetcher:    fetcher,
	}, nil
}

// Handle implements the renderer
func (r *sitemapRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	var routeIndex int = -1
	for index, route := range r.config.Routes {
		if route.Path != req.URL.Path {
			continue
		}

		routeIndex = index

		break
	}
	if routeIndex == -1 {
		r.next.Handle(w, req, info)

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
		err = r.sitemapIndex(&r.config.Routes[routeIndex].SitemapIndex, req, w)
	case sitemapKindSitemap:
		err = r.sitemap(&r.config.Routes[routeIndex].Sitemap, req, w)
	}
	if err != nil {
		r.logger.Printf("Failed to render: %s", err)

		return err
	}

	return nil
}

// sitemapIndex generates a sitemap index
func (r *sitemapRenderer) sitemapIndex(s *[]SitemapIndexEntry, req *http.Request, w io.Writer) error {
	w.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n"))
	w.Write([]byte("<sitemapindex xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n"))

	var err error
	for _, item := range *s {
		switch item.Type {
		case sitemapEntrySitemapIndexTypeStatic:
			err = r.sitemapIndexStatic(&item.Static, req, w)
		}
		if err != nil {
			return err
		}
	}

	w.Write([]byte("</sitemapindex>\n"))

	return nil
}

// sitemapIndexStatic generates a sitemap index static entry
func (r *sitemapRenderer) sitemapIndexStatic(static *SitemapIndexEntryStatic, req *http.Request,
	w io.Writer) error {
	w.Write([]byte("<sitemap>\n"))
	w.Write([]byte(fmt.Sprintf("<loc>%s</loc>\n", r.sitemapAbsLink(static.Loc, r.config.Root))))
	w.Write([]byte("</sitemap>\n"))

	return nil
}

// sitemap generates a sitemap
func (r *sitemapRenderer) sitemap(s *[]SitemapEntry, req *http.Request, w io.Writer) error {
	w.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n"))
	w.Write([]byte("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\"\n"))
	w.Write([]byte("   xmlns:xhtml=\"http://www.w3.org/1999/xhtml\">\n"))

	var err error
	for _, item := range *s {
		switch item.Type {
		case sitemapEntrySitemapTypeStatic:
			err = r.sitemapStatic(&item.Static, req, w)
		case sitemapEntrySitemapTypeList:
			err = r.sitemapList(&item.List, req, w)
		}
		if err != nil {
			return err
		}
	}

	w.Write([]byte("</urlset>\n"))

	return nil
}

// sitemapStatic generates a sitemap static entry
func (r *sitemapRenderer) sitemapStatic(static *SitemapEntryStatic, req *http.Request, w io.Writer) error {
	w.Write([]byte("<url>\n"))
	w.Write([]byte(fmt.Sprintf("<loc>%s</loc>\n", r.sitemapAbsLink(static.Loc, r.config.Root))))
	if static.Lastmod != nil {
		w.Write([]byte(fmt.Sprintf("<lastmod>%s</lastmod>\n", *static.Lastmod)))
	}
	if static.Changefreq != nil {
		w.Write([]byte(fmt.Sprintf("<changefreq>%s</changefreq>\n", *static.Changefreq)))
	}
	if static.Priority != nil {
		w.Write([]byte(fmt.Sprintf("<priority>%.1f</priority>\n", *static.Priority)))
	}
	w.Write([]byte("</url>\n"))

	return nil
}

// sitemapList generates a sitemap list entry
func (r *sitemapRenderer) sitemapList(list *SitemapEntryList, req *http.Request, w io.Writer) error {
	response, err := r.fetcher.Get(list.Resource)
	if err != nil {
		return nil
	}

	var payload interface{}
	err = json.Unmarshal(response, &payload)
	if err != nil {
		return err
	}
	mPayload := payload.(map[string]interface{})
	responseData := mPayload[list.ResourcePayloadItems]
	payloadDataArray := responseData.([]interface{})
	for _, item := range payloadDataArray {
		mItem := item.(map[string]interface{})

		var loc, lastmod string
		if v, ok := mItem[list.ResourcePayloadItemLoc].(string); ok {
			loc = v
		} else {
			continue
		}
		if list.ResourcePayloadItemLastmod != nil {
			if v, ok := mItem[*list.ResourcePayloadItemLastmod].(string); ok {
				lastmod = v
			}
		}

		w.Write([]byte("<url>\n"))
		w.Write([]byte(fmt.Sprintf("<loc>%s</loc>\n", r.sitemapAbsLink(loc, r.config.Root))))
		if lastmod != "" {
			w.Write([]byte(fmt.Sprintf("<lastmod>%s</lastmod>\n", lastmod)))
		}
		if list.Changefreq != nil {
			w.Write([]byte(fmt.Sprintf("<changefreq>%s</changefreq>\n", *list.Changefreq)))
		}
		if list.Priority != nil {
			w.Write([]byte(fmt.Sprintf("<priority>%.1f</priority>\n", *list.Priority)))
		}
		w.Write([]byte("</url>\n"))
	}

	return nil
}

// sitemapAbsLink returns the absolute address of the given link
func (r *sitemapRenderer) sitemapAbsLink(link string, root string) string {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}
	return fmt.Sprintf("%s%s", root, link)
}
