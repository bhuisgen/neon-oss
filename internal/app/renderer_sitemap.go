// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	result, err := r.render(routeIndex, req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, http.StatusInternalServerError)

		return
	}

	w.WriteHeader(result.Status)
	w.Write(result.Body)

	r.logger.Printf("Render completed (url=%s, status=%d, valid=%t, cache=%t)", req.URL.Path, result.Status, result.Valid,
		result.Cache)
}

// Next configures the next renderer
func (r *sitemapRenderer) Next(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *sitemapRenderer) render(routeIndex int, req *http.Request) (*Render, error) {
	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*Render)

			return result, nil
		}
	}

	var body []byte
	var state bool
	var err error
	switch r.config.Routes[routeIndex].Kind {
	case sitemapKindSitemapIndex:
		body, state, err = sitemapIndex(&r.config.Routes[routeIndex].SitemapIndex, r, req)
	case sitemapKindSitemap:
		body, state, err = sitemap(&r.config.Routes[routeIndex].Sitemap, r, req)
	}
	if err != nil {
		r.logger.Printf("Failed to render: %s", err)

		return nil, err
	}

	var valid bool = true
	var status int = http.StatusOK
	if !state {
		valid = false
	}
	if !valid {
		status = http.StatusServiceUnavailable
	}

	result := Render{
		Body:   body,
		Valid:  valid,
		Status: status,
	}
	if result.Valid && r.config.Cache {
		r.cache.Set(req.URL.Path, &result, time.Duration(r.config.CacheTTL)*time.Second)
		result.Cache = true
	}

	return &result, nil
}

// sitemapIndex generates a sitemap index
func sitemapIndex(s *[]SitemapIndexEntry, r *sitemapRenderer, req *http.Request) ([]byte, bool, error) {
	body := r.bufferPool.Get()
	defer r.bufferPool.Put(body)

	body.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n")
	body.WriteString("<sitemapindex xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	var valid bool = true
	var state bool
	var err error
	for _, item := range *s {
		switch item.Type {
		case sitemapEntrySitemapIndexTypeStatic:
			state, err = sitemapIndexStatic(&item.Static, r, req, body)
		}
		if err != nil {
			return nil, false, err
		}
		if !state {
			valid = false
		}
	}

	body.WriteString("</sitemapindex>\n")

	return body.Bytes(), valid, err
}

// sitemapIndexStatic generates a sitemap index static entry
func sitemapIndexStatic(static *SitemapIndexEntryStatic, r *sitemapRenderer, req *http.Request,
	buf *bytes.Buffer) (bool, error) {
	buf.WriteString("<sitemap>\n")
	buf.WriteString(fmt.Sprintf("<loc>%s</loc>\n", sitemapAbsLink(static.Loc, r.config.Root)))
	buf.WriteString("</sitemap>\n")

	return true, nil
}

// sitemap generates a sitemap
func sitemap(s *[]SitemapEntry, r *sitemapRenderer, req *http.Request) ([]byte, bool, error) {
	body := r.bufferPool.Get()
	defer r.bufferPool.Put(body)

	body.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n")
	body.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\"\n")
	body.WriteString("   xmlns:xhtml=\"http://www.w3.org/1999/xhtml\">\n")

	var valid bool = true
	var state bool
	var err error
	for _, item := range *s {
		switch item.Type {
		case sitemapEntrySitemapTypeStatic:
			state, err = sitemapStatic(&item.Static, r, req, body)
		case sitemapEntrySitemapTypeList:
			state, err = sitemapList(&item.List, r, req, body)
		}
		if err != nil {
			return nil, false, err
		}
		if !state {
			valid = false
		}
	}

	body.WriteString("</urlset>\n")

	return body.Bytes(), valid, err
}

// sitemapStatic generates a sitemap static entry
func sitemapStatic(static *SitemapEntryStatic, r *sitemapRenderer, req *http.Request,
	buf *bytes.Buffer) (bool, error) {
	buf.WriteString("<url>\n")
	buf.WriteString(fmt.Sprintf("<loc>%s</loc>\n", sitemapAbsLink(static.Loc, r.config.Root)))
	if static.Lastmod != nil {
		buf.WriteString(fmt.Sprintf("<lastmod>%s</lastmod>\n", *static.Lastmod))
	}
	if static.Changefreq != nil {
		buf.WriteString(fmt.Sprintf("<changefreq>%s</changefreq>\n", *static.Changefreq))
	}
	if static.Priority != nil {
		buf.WriteString(fmt.Sprintf("<priority>%.1f</priority>\n", *static.Priority))
	}
	buf.WriteString("</url>\n")

	return true, nil
}

// sitemapList generates a sitemap list entry
func sitemapList(list *SitemapEntryList, r *sitemapRenderer, req *http.Request,
	buf *bytes.Buffer) (bool, error) {
	response, err := r.fetcher.Get(list.Resource)
	if err != nil {
		return false, nil
	}

	var payload interface{}
	err = json.Unmarshal(response, &payload)
	if err != nil {
		return false, err
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

		buf.WriteString("<url>\n")
		buf.WriteString(fmt.Sprintf("<loc>%s</loc>\n", sitemapAbsLink(loc, r.config.Root)))
		if lastmod != "" {
			buf.WriteString(fmt.Sprintf("<lastmod>%s</lastmod>\n", lastmod))
		}
		if list.Changefreq != nil {
			buf.WriteString(fmt.Sprintf("<changefreq>%s</changefreq>\n", *list.Changefreq))
		}
		if list.Priority != nil {
			buf.WriteString(fmt.Sprintf("<priority>%.1f</priority>\n", *list.Priority))
		}
		buf.WriteString("</url>\n")
	}

	return true, nil
}

// sitemapAbsLink returns the absolute address of the given link
func sitemapAbsLink(link string, root string) string {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}
	return fmt.Sprintf("%s%s", root, link)
}
