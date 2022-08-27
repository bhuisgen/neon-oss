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
	"strings"
	"time"
)

const (
	SITEMAP_KIND_SITEMAP                   string = "sitemap"
	SITEMAP_KIND_SITEMAPINDEX              string = "sitemapindex"
	SITEMAP_ENTRY_SITEMAPINDEX_TYPE_STATIC string = "static"
	SITEMAP_ENTRY_SITEMAP_TYPE_STATIC      string = "static"
	SITEMAP_ENTRY_SITEMAP_TYPE_LIST        string = "list"
)

// sitemapRenderer implements the sitemap renderer
type sitemapRenderer struct {
	Renderer
	next Renderer

	config  *SitemapRendererConfig
	logger  *log.Logger
	cache   *cache
	fetcher *fetcher
}

// SitemapRendererConfig implements the sitemap renderer configuration
type SitemapRendererConfig struct {
	Enable   bool
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
	Lastmod    string
	Changefreq string
	Priority   string
}

// SitemapEntryList implements a sitemap entry list
type SitemapEntryList struct {
	Resource                   string
	ResourcePayloadItems       string
	ResourcePayloadItemLoc     string
	ResourcePayloadItemLastmod string
	Changefreq                 string
	Priority                   string
}

// CreateSitemapRenderer creates a new sitemap renderer
func CreateSitemapRenderer(config *SitemapRendererConfig, fetcher *fetcher) (*sitemapRenderer, error) {
	return &sitemapRenderer{
		config:  config,
		logger:  log.Default(),
		cache:   NewCache(),
		fetcher: fetcher,
	}, nil
}

// handle implements the renderer handler
func (r *sitemapRenderer) handle(w http.ResponseWriter, req *http.Request) {
	if !r.config.Enable {
		r.next.handle(w, req)

		return
	}

	var routeIndex int = -1
	for index, route := range r.config.Routes {
		if route.Path != req.URL.Path {
			continue
		}

		routeIndex = index

		break
	}
	if routeIndex == -1 {
		r.next.handle(w, req)

		return
	}

	result, err := r.render(routeIndex, req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		return
	}

	w.WriteHeader(result.Status)
	w.Write(result.Body)

	r.logger.Printf("Render completed (url=%s, status=%d, valid=%t, cache=%t)", req.URL.Path, result.Status, result.Valid,
		result.Cache)
}

// setNext configures the next renderer
func (r *sitemapRenderer) setNext(renderer Renderer) {
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

	var valid bool = true
	var status = http.StatusOK
	var body = bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(body)
	body.Reset()

	var state bool
	var err error
	switch r.config.Routes[routeIndex].Kind {
	case SITEMAP_KIND_SITEMAPINDEX:
		state, err = r.generateSitemapIndex(routeIndex, body)
	case SITEMAP_KIND_SITEMAP:
		state, err = r.generateSitemap(routeIndex, body)
	}
	if !state || err != nil {
		valid = false
	}

	if !valid {
		status = http.StatusServiceUnavailable
	}

	result := Render{
		Body:   body.Bytes(),
		Status: status,
		Valid:  valid,
		Cache:  r.config.Cache,
	}

	if result.Valid && r.config.Cache {
		r.cache.Set(req.URL.Path, &result, time.Duration(r.config.CacheTTL)*time.Second)
	}

	return &result, nil
}

// generateSitemapIndex generates a sitemap index
func (r *sitemapRenderer) generateSitemapIndex(routeIndex int, body *bytes.Buffer) (bool, error) {
	body.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n"))
	body.Write([]byte("<sitemapindex xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n"))

	var valid bool = true

	var state bool
	var err error
	for _, item := range r.config.Routes[routeIndex].SitemapIndex {
		switch item.Type {
		case SITEMAP_ENTRY_SITEMAPINDEX_TYPE_STATIC:
			state, err = r.generateSitemapIndexStatic(body, r.absLink(item.Static.Loc))
		}
	}
	if !state || err != nil {
		valid = false
	}

	body.Write([]byte("</sitemapindex>\n"))

	return valid, nil
}

// generateSitemapIndexStatic generates a sitemap index static item
func (r *sitemapRenderer) generateSitemapIndexStatic(buf *bytes.Buffer, loc string) (bool, error) {
	buf.Write([]byte("<sitemap>\n"))
	buf.Write([]byte(fmt.Sprintf("<loc>%s</loc>\n", loc)))
	buf.Write([]byte("</sitemap>\n"))

	return true, nil
}

// generateSitemap generates a sitemap
func (r *sitemapRenderer) generateSitemap(routeIndex int, body *bytes.Buffer) (bool, error) {
	body.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n"))
	body.Write([]byte("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\"\n"))
	body.Write([]byte("	 xmlns:xhtml=\"http://www.w3.org/1999/xhtml\">\n"))

	var valid bool = true

	var state bool
	var err error
	for _, item := range r.config.Routes[routeIndex].Sitemap {
		switch item.Type {
		case SITEMAP_ENTRY_SITEMAP_TYPE_STATIC:
			state, err = r.generateSitemapStatic(body, r.absLink(item.Static.Loc), item.Static.Lastmod,
				item.Static.Changefreq, item.Static.Priority)
		case SITEMAP_ENTRY_SITEMAP_TYPE_LIST:
			state, err = r.generateSitemapList(body, item.List.Resource, item.List.ResourcePayloadItems,
				item.List.ResourcePayloadItemLoc, item.List.ResourcePayloadItemLastmod, item.List.Changefreq,
				item.List.Priority)
		}
		if err != nil || !state {
			valid = false
		}
	}

	body.Write([]byte("</urlset>\n"))

	return valid, nil
}

// generateSitemapStatic generates a sitemap static item
func (r *sitemapRenderer) generateSitemapStatic(buf *bytes.Buffer, loc string, lastmod string,
	changefreq string, priority string) (bool, error) {
	buf.Write([]byte("<url>\n"))
	buf.Write([]byte(fmt.Sprintf("<loc>%s</loc>\n", loc)))
	buf.Write([]byte(fmt.Sprintf("<lastmod>%s</lastmod>\n", lastmod)))
	buf.Write([]byte(fmt.Sprintf("<changefreq>%s</changefreq>\n", changefreq)))
	buf.Write([]byte(fmt.Sprintf("<priority>%s</priority>\n", priority)))
	buf.Write([]byte("</url>\n"))

	return true, nil
}

// generateSitemapStatic generates a sitemap list item
func (r *sitemapRenderer) generateSitemapList(buf *bytes.Buffer, resource string, payloadItems string,
	payloadItemLoc string, payloadItemLastmod string, changefreq string, priority string) (bool, error) {
	response, err := r.fetcher.Get(resource)
	if err != nil {
		return false, nil
	}

	var payload interface{}

	err = json.Unmarshal(response, &payload)
	if err != nil {
		return false, err
	}

	mPayload := payload.(map[string]interface{})
	responseData := mPayload[payloadItems]
	payloadDataArray := responseData.([]interface{})

	if len(payloadDataArray) > 0 {
		for _, item := range payloadDataArray {
			mItem := item.(map[string]interface{})

			buf.Write([]byte("<url>\n"))
			buf.Write([]byte(fmt.Sprintf("<loc>%s</loc>\n", r.absLink(mItem[payloadItemLoc].(string)))))
			buf.Write([]byte(fmt.Sprintf("<lastmod>%s</lastmod>\n", mItem[payloadItemLastmod].(string))))
			buf.Write([]byte(fmt.Sprintf("<changefreq>%s</changefreq>\n", changefreq)))
			buf.Write([]byte(fmt.Sprintf("<priority>%s</priority>\n", priority)))
			buf.Write([]byte("</url>\n"))
		}
	}

	return true, nil
}

// absLink returns the absolute address of the given link
func (r *sitemapRenderer) absLink(link string) string {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}

	return fmt.Sprintf("%s%s", r.config.Root, link)
}
