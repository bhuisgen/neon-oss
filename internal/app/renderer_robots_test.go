// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"testing"
	"text/template"
)

type testRobotsRendererNextRenderer struct{}

func (r testRobotsRendererNextRenderer) Handle(w http.ResponseWriter, req *http.Request, i *ServerInfo) {
}

func (r testRobotsRendererNextRenderer) Next(renderer Renderer) {
}

type testRobotsRendererResponseWriter struct {
	header http.Header
}

func (w testRobotsRendererResponseWriter) Header() http.Header {
	return w.header
}

func (w testRobotsRendererResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w testRobotsRendererResponseWriter) WriteHeader(statusCode int) {
}

func TestCreateRobotsRenderer(t *testing.T) {
	type args struct {
		config *RobotsRendererConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				config: &RobotsRendererConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateRobotsRenderer(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRobotsRenderer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRobotsRendererHandle(t *testing.T) {
	type fields struct {
		config     *RobotsRendererConfig
		logger     *log.Logger
		template   *template.Template
		bufferPool BufferPool
		cache      Cache
		next       Renderer
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
			name: "default",
			fields: fields{
				config: &RobotsRendererConfig{
					Path: "/robots.txt",
				},
				logger:     log.Default(),
				template:   template.Must(template.New("robots").Parse(robotsTemplate)),
				bufferPool: newBufferPool(),
				cache:      newCache(),
				next:       testRobotsRendererNextRenderer{},
			},
			args: args{
				w: testRobotsRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/robots.txt",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "cache",
			fields: fields{
				config: &RobotsRendererConfig{
					Path:     "/robots.txt",
					Cache:    true,
					CacheTTL: 60,
				},
				logger:     log.Default(),
				template:   template.Must(template.New("robots").Parse(robotsTemplate)),
				bufferPool: newBufferPool(),
				cache:      newCache(),
			},
			args: args{
				w: testRobotsRendererResponseWriter{},
				req: &http.Request{
					URL: &url.URL{
						Path: "/robots.txt",
					},
				},
				info: &ServerInfo{},
			},
		},
		{
			name: "next passthrough",
			fields: fields{
				config: &RobotsRendererConfig{
					Path: "/robots.txt",
				},
				logger:     log.Default(),
				template:   template.Must(template.New("robots").Parse(robotsTemplate)),
				bufferPool: newBufferPool(),
				cache:      newCache(),
				next:       testRobotsRendererNextRenderer{},
			},
			args: args{
				w: testRobotsRendererResponseWriter{},
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
			r := &robotsRenderer{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				template:   tt.fields.template,
				bufferPool: tt.fields.bufferPool,
				cache:      tt.fields.cache,
				next:       tt.fields.next,
			}
			r.Handle(tt.args.w, tt.args.req, tt.args.info)
		})
	}
}

func TestRobotsRendererNext(t *testing.T) {
	type fields struct {
		config     *RobotsRendererConfig
		logger     *log.Logger
		template   *template.Template
		bufferPool BufferPool
		next       Renderer
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
			r := &robotsRenderer{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				template:   tt.fields.template,
				bufferPool: tt.fields.bufferPool,
				next:       tt.fields.next,
			}
			r.Next(tt.args.renderer)
		})
	}
}

func TestRobotsRendererRender(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	type fields struct {
		config     *RobotsRendererConfig
		logger     *log.Logger
		template   *template.Template
		bufferPool BufferPool
		cache      Cache
		next       Renderer
	}
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "required",
			fields: fields{
				config: &RobotsRendererConfig{
					Path: "/robots.txt",
				},
				logger:     log.Default(),
				template:   template.Must(template.New("robots").Parse(robotsTemplate)),
				bufferPool: newBufferPool(),
				cache:      newCache(),
			},
			args: args{
				req: &http.Request{
					URL: &url.URL{
						Path: "/robots.txt",
					},
				},
			},
			wantW: `User-agent: *
Disallow: /
`,
		},
		{
			name: "allow host",
			fields: fields{
				config: &RobotsRendererConfig{
					Path:  "/robots.txt",
					Hosts: []string{"localhost", "test"},
				},
				logger:     log.Default(),
				template:   template.Must(template.New("robots").Parse(robotsTemplate)),
				bufferPool: newBufferPool(),
				cache:      newCache(),
			},
			args: args{
				req: &http.Request{
					Host: "test",
					URL: &url.URL{
						Path: "/robots.txt",
					},
				},
			},
			wantW: `User-agent: *
Allow: /
`,
		},
		{
			name: "disallow host",
			fields: fields{
				config: &RobotsRendererConfig{
					Path:  "/robots.txt",
					Hosts: []string{"localhost"},
				},
				logger:     log.Default(),
				template:   template.Must(template.New("robots").Parse(robotsTemplate)),
				bufferPool: newBufferPool(),
				cache:      newCache(),
			},
			args: args{
				req: &http.Request{
					Host: "test",
					URL: &url.URL{
						Path: "/robots.txt",
					},
				},
			},
			wantW: `User-agent: *
Disallow: /
`,
		},
		{
			name: "sitemap",
			fields: fields{
				config: &RobotsRendererConfig{
					Path:     "/robots.txt",
					Hosts:    []string{"localhost"},
					Sitemaps: []string{"http://localhost/sitemap1.xml", "http://localhost/sitemap2.xml"},
				},
				logger:     log.Default(),
				template:   template.Must(template.New("robots").Parse(robotsTemplate)),
				bufferPool: newBufferPool(),
				cache:      newCache(),
			},
			args: args{
				req: &http.Request{
					Host: "localhost",
					URL: &url.URL{
						Path: "/robots.txt",
					},
				},
			},
			wantW: `User-agent: *
Allow: /

Sitemap: http://localhost/sitemap1.xml
Sitemap: http://localhost/sitemap2.xml
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &robotsRenderer{
				config:     tt.fields.config,
				logger:     tt.fields.logger,
				template:   tt.fields.template,
				bufferPool: tt.fields.bufferPool,
				cache:      tt.fields.cache,
				next:       tt.fields.next,
			}
			w := &bytes.Buffer{}
			if err := r.render(tt.args.req, w); (err != nil) != tt.wantErr {
				t.Errorf("robotsRenderer.render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("robotsRenderer.render() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}