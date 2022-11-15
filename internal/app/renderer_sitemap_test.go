// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"testing"
	"text/template"
)

type testSitemapRendererNextRenderer struct{}

func (r testSitemapRendererNextRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
}

func (r testSitemapRendererNextRenderer) Next(renderer Renderer) {
}

type testSitemapRendererResponseWriter struct {
	header http.Header
}

func (w testSitemapRendererResponseWriter) Header() http.Header {
	return w.header
}

func (w testSitemapRendererResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testSitemapRendererResponseWriter) WriteHeader(statusCode int) {
}

type testSitemapRendererFetcher struct {
	errFetch                           bool
	errFetchOnlyForNames               []string
	exists                             bool
	get                                []byte
	errGet                             bool
	createResourceFromTemplateResource *Resource
	errCreateResourceFromTemplate      bool
}

func (t testSitemapRendererFetcher) Fetch(ctx context.Context, name string) error {
	if t.errFetch {
		return errors.New("test error")
	}
	for _, n := range t.errFetchOnlyForNames {
		if n == name {
			return errors.New("test error")
		}
	}
	return nil
}

func (t testSitemapRendererFetcher) Exists(name string) bool {
	return t.exists
}

func (t testSitemapRendererFetcher) Get(name string) ([]byte, error) {
	if t.errGet {
		return nil, errors.New("test error")
	}
	return t.get, nil
}

func (t testSitemapRendererFetcher) Register(r Resource) {
}

func (t testSitemapRendererFetcher) Unregister(name string) {
}

func (t testSitemapRendererFetcher) CreateResourceFromTemplate(template string, resource string,
	params map[string]string, headers map[string]string) (*Resource, error) {
	if t.errCreateResourceFromTemplate {
		return nil, errors.New("test error")
	}
	return t.createResourceFromTemplateResource, nil
}

func TestCreateSitemapRenderer(t *testing.T) {
	type args struct {
		config  *SitemapRendererConfig
		fetcher Fetcher
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &SitemapRendererConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateSitemapRenderer(tt.args.config, tt.args.fetcher)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSitemapRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestSitemapRendererHandle(t *testing.T) {
	type fields struct {
		config               *SitemapRendererConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		bufferPool           BufferPool
		cache                Cache
		fetcher              Fetcher
		next                 Renderer
	}
	type args struct {
		w    http.ResponseWriter
		req  *http.Request
		info *ServerInfo
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "sitemap index",
			fields: fields{
				config: &SitemapRendererConfig{
					Root: "http://localhost",
					Routes: []SitemapRoute{
						{
							Path: "/sitemap.xml",
							Kind: "sitemap",
							Sitemap: []SitemapEntry{
								{
									Name: "static",
									Type: "static",
									Static: SitemapEntryStatic{
										Loc: "/",
									},
								},
							},
						},
					},
				},
				logger:               log.Default(),
				templateSitemapIndex: template.Must(template.New("sitemap_index").Parse(sitemapTemplateSitemapIndex)),
				templateSitemap:      template.Must(template.New("sitemap").Parse(sitemapTemplateSitemap)),
				bufferPool:           newBufferPool(),
				cache:                newCache(),
				next:                 testSitemapRendererNextRenderer{},
			},
			args: args{
				w: testSitemapRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/sitemap.xml",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "sitemap",
			fields: fields{
				config: &SitemapRendererConfig{
					Root: "http://localhost",
					Routes: []SitemapRoute{
						{
							Path: "/sitemap.xml",
							Kind: "sitemap",
							Sitemap: []SitemapEntry{
								{
									Name: "static",
									Type: "static",
									Static: SitemapEntryStatic{
										Loc: "/",
									},
								},
							},
						},
					},
				},
				logger:               log.Default(),
				templateSitemapIndex: template.Must(template.New("sitemap_index").Parse(sitemapTemplateSitemapIndex)),
				templateSitemap:      template.Must(template.New("sitemap").Parse(sitemapTemplateSitemap)),
				bufferPool:           newBufferPool(),
				cache:                newCache(),
				next:                 testSitemapRendererNextRenderer{},
			},
			args: args{
				w: testSitemapRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/sitemap.xml",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "sitemap with cache",
			fields: fields{
				config: &SitemapRendererConfig{
					Root:     "http://localhost",
					Cache:    true,
					CacheTTL: 60,
					Routes: []SitemapRoute{
						{
							Path: "/sitemap.xml",
							Kind: "sitemap",
							Sitemap: []SitemapEntry{
								{
									Name: "static",
									Type: "static",
									Static: SitemapEntryStatic{
										Loc: "/",
									},
								},
							},
						},
					},
				},
				logger:               log.Default(),
				templateSitemapIndex: template.Must(template.New("sitemap_index").Parse(sitemapTemplateSitemapIndex)),
				templateSitemap:      template.Must(template.New("sitemap").Parse(sitemapTemplateSitemap)),
				bufferPool:           newBufferPool(),
				cache:                newCache(),
				next:                 testSitemapRendererNextRenderer{},
			},
			args: args{
				w: testSitemapRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/sitemap.xml",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "next passthrough",
			fields: fields{
				config: &SitemapRendererConfig{
					Root: "http://localhost",
					Routes: []SitemapRoute{
						{
							Path: "/sitemap.xml",
							Kind: "sitemap",
							Sitemap: []SitemapEntry{
								{
									Name: "static",
									Type: "static",
									Static: SitemapEntryStatic{
										Loc: "/",
									},
								},
							},
						},
					},
				},
				logger:               log.Default(),
				templateSitemapIndex: template.Must(template.New("sitemap_index").Parse(sitemapTemplateSitemapIndex)),
				templateSitemap:      template.Must(template.New("sitemap").Parse(sitemapTemplateSitemap)),
				bufferPool:           newBufferPool(),
				cache:                newCache(),
				next:                 testSitemapRendererNextRenderer{},
			},
			args: args{
				w: testSitemapRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/index.html",
					},
				},
				info: &ServerInfo{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &sitemapRenderer{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				bufferPool:           tt.fields.bufferPool,
				cache:                tt.fields.cache,
				fetcher:              tt.fields.fetcher,
				next:                 tt.fields.next,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestSitemapRendererNext(t *testing.T) {
	type fields struct {
		config               *SitemapRendererConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		bufferPool           BufferPool
		cache                Cache
		fetcher              Fetcher
		next                 Renderer
	}
	type args struct {
		renderer Renderer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &sitemapRenderer{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				bufferPool:           tt.fields.bufferPool,
				cache:                tt.fields.cache,
				fetcher:              tt.fields.fetcher,
				next:                 tt.fields.next,
			}
			r.Next(tt.args.renderer)
		})
	}
}

func TestSitemapRendererRender(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	type fields struct {
		config               *SitemapRendererConfig
		logger               *log.Logger
		templateSitemapIndex *template.Template
		templateSitemap      *template.Template
		bufferPool           BufferPool
		cache                Cache
		fetcher              Fetcher
		next                 Renderer
	}
	type args struct {
		routeIndex int
		req        *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "sitemap index",
			fields: fields{
				config: &SitemapRendererConfig{
					Root: "http://localhost",
					Routes: []SitemapRoute{
						{
							Path: "/sitemap.xml",
							Kind: "sitemap_index",
							SitemapIndex: []SitemapIndexEntry{
								{
									Name: "static",
									Type: "static",
									Static: SitemapIndexEntryStatic{
										Loc: "/",
									},
								},
							},
						},
					},
				},
				logger:               log.Default(),
				templateSitemapIndex: template.Must(template.New("sitemap_index").Parse(sitemapTemplateSitemapIndex)),
				bufferPool:           newBufferPool(),
				cache:                newCache(),
				fetcher:              &testSitemapRendererFetcher{},
			},
			args: args{
				routeIndex: 0,
				req: &http.Request{
					URL: &url.URL{
						Path: "/sitemap.xml",
					},
				},
			},
			wantW: `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<sitemap>
<loc>http://localhost/</loc>
</sitemap>
</sitemapindex>
`,
		},
		{
			name: "sitemap static",
			fields: fields{
				config: &SitemapRendererConfig{
					Root: "http://localhost",
					Routes: []SitemapRoute{
						{
							Path: "/sitemap.xml",
							Kind: "sitemap",
							Sitemap: []SitemapEntry{
								{
									Name: "static",
									Type: "static",
									Static: SitemapEntryStatic{
										Loc:        "/",
										Lastmod:    stringPtr("2022-10-14T12:00:00.000Z"),
										Changefreq: stringPtr("daily"),
										Priority:   floatPtr(0.5),
									},
								},
							},
						},
					},
				},
				logger:          log.Default(),
				templateSitemap: template.Must(template.New("sitemap").Parse(sitemapTemplateSitemap)),
				bufferPool:      newBufferPool(),
				cache:           newCache(),
				fetcher:         &testSitemapRendererFetcher{},
			},
			args: args{
				routeIndex: 0,
				req: &http.Request{
					URL: &url.URL{
						Path: "/sitemap.xml",
					},
				},
			},
			wantW: `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
   xmlns:xhtml="http://www.w3.org/1999/xhtml">
<url>
<loc>http://localhost/</loc>
<lastmod>2022-10-14T12:00:00.000Z</lastmod>
<changefreq>daily</changefreq>
<priority>0.5</priority>
</url>
</urlset>
`,
		},
		{
			name: "sitemap list",
			fields: fields{
				config: &SitemapRendererConfig{
					Root: "http://localhost",
					Routes: []SitemapRoute{
						{
							Path: "/sitemap.xml",
							Kind: "sitemap",
							Sitemap: []SitemapEntry{
								{
									Name: "list",
									Type: "list",
									List: SitemapEntryList{
										Resource:                   "resource",
										ResourcePayloadItems:       "data",
										ResourcePayloadItemLoc:     "loc",
										ResourcePayloadItemLastmod: stringPtr("lastmod"),
										ResourcePayloadItemIgnore:  stringPtr("ignore"),
										Changefreq:                 stringPtr("daily"),
										Priority:                   floatPtr(0.5),
									},
								},
							},
						},
					},
				},
				logger:          log.Default(),
				templateSitemap: template.Must(template.New("sitemap").Parse(sitemapTemplateSitemap)),
				bufferPool:      newBufferPool(),
				cache:           newCache(),
				fetcher: &testSitemapRendererFetcher{
					get: []byte(`{"data": [
								{
									"id": 1,
									"loc": "/item1",
									"lastmod": "2022-10-15T12:00:00.000Z"
								},
								{
									"id": 2,
									"loc": "/item2",
									"lastmod": "2022-10-14T12:00:00.000Z"
								},
								{
									"id": 3,
									"loc": "/item3",
									"lastmod": "2022-10-14T12:00:00.000Z",
									"ignore": true
								}
							]
						}`),
				},
			},
			args: args{
				routeIndex: 0,
				req: &http.Request{
					URL: &url.URL{
						Path: "/sitemap.xml",
					},
				},
			},
			wantW: `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
   xmlns:xhtml="http://www.w3.org/1999/xhtml">
<url>
<loc>http://localhost/item1</loc>
<lastmod>2022-10-15T12:00:00.000Z</lastmod>
<changefreq>daily</changefreq>
<priority>0.5</priority>
</url>
<url>
<loc>http://localhost/item2</loc>
<lastmod>2022-10-14T12:00:00.000Z</lastmod>
<changefreq>daily</changefreq>
<priority>0.5</priority>
</url>
</urlset>
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &sitemapRenderer{
				config:               tt.fields.config,
				logger:               tt.fields.logger,
				templateSitemapIndex: tt.fields.templateSitemapIndex,
				templateSitemap:      tt.fields.templateSitemap,
				bufferPool:           tt.fields.bufferPool,
				cache:                tt.fields.cache,
				fetcher:              tt.fields.fetcher,
				next:                 tt.fields.next,
			}
			w := &bytes.Buffer{}
			if err := r.render(tt.args.routeIndex, tt.args.req, w); (err != nil) != tt.wantErr {
				t.Errorf("sitemapRenderer.render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("sitemapRenderer.render() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
