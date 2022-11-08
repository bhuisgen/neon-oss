// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// indexResourceResult implements the results of a resource
type indexResourceResult struct {
	Loading  bool   `json:"loading"`
	Error    string `json:"error"`
	Response string `json:"response"`
}

// indexRenderer implements the index renderer
type indexRenderer struct {
	config      *IndexRendererConfig
	logger      *log.Logger
	regexps     []*regexp.Regexp
	html        *[]byte
	htmlInfo    *time.Time
	bundle      *string
	bundleInfo  *time.Time
	bufferPool  BufferPool
	vmPool      VMPool
	cache       Cache
	fetcher     Fetcher
	next        Renderer
	osStat      func(name string) (fs.FileInfo, error)
	osReadFile  func(name string) ([]byte, error)
	jsonMarshal func(v any) ([]byte, error)
}

// IndexRendererConfig implements the index renderer configuration
type IndexRendererConfig struct {
	HTML      string
	Bundle    *string
	Env       string
	Container string
	State     string
	Timeout   int
	MaxVMs    int
	Cache     bool
	CacheTTL  int
	Rules     []IndexRule
}

// IndexRule implements a rule
type IndexRule struct {
	Path  string
	State []IndexRuleStateEntry
	Last  bool
}

// IndexRuleStateEntry implements a rule state entry
type IndexRuleStateEntry struct {
	Key      string
	Resource string
	Export   *bool
}

// indexRender implements a render
type indexRender struct {
	Body           []byte
	Status         int
	Redirect       bool
	RedirectURL    string
	RedirectStatus int
	Headers        map[string]string
}

// indexPage implements a page
type indexPage struct {
	HTML    *[]byte
	Render  *[]byte
	State   *string
	Title   *string
	Metas   *domElementList
	Links   *domElementList
	Scripts *domElementList
}

const (
	indexLogger           string = "server[index]"
	indexResourceNotExist string = "unknown resource"
)

// indexOsStat redirects to os.Stat
func indexOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// indexOsReadFile redirects to os.ReadFile
func indexOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// indexJsonMarshal redirects to json.Marshal
func indexJsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// CreateIndexRenderer creates a new index renderer
func CreateIndexRenderer(config *IndexRendererConfig, fetcher Fetcher) (*indexRenderer, error) {
	r := indexRenderer{
		config:      config,
		logger:      log.New(os.Stderr, fmt.Sprint(indexLogger, ": "), log.LstdFlags|log.Lmsgprefix),
		regexps:     []*regexp.Regexp{},
		bufferPool:  newBufferPool(),
		vmPool:      newVMPool(int32(config.MaxVMs)),
		cache:       newCache(),
		fetcher:     fetcher,
		osStat:      indexOsStat,
		osReadFile:  indexOsReadFile,
		jsonMarshal: indexJsonMarshal,
	}

	err := r.initialize()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// initialize initializes the renderer
func (r *indexRenderer) initialize() error {
	for _, rule := range r.config.Rules {
		re, err := regexp.Compile(rule.Path)
		if err != nil {
			return err
		}
		r.regexps = append(r.regexps, re)
	}

	return nil
}

// Handle implements the renderer
func (r *indexRenderer) Handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*indexRender)

			for _, header := range result.Headers {
				w.Header().Add(header, result.Headers[header])
			}

			if result.Redirect {
				http.Redirect(w, req, result.RedirectURL, result.RedirectStatus)

				r.logger.Printf("Render completed (url=%s, redirect=%s, status=%d, cache=%t)", req.URL.Path, result.RedirectURL,
					result.RedirectStatus, true)

				return
			}

			w.WriteHeader(result.Status)
			w.Write(result.Body)

			r.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", req.URL.Path, result.Status, true)

			return
		}
	}

	b := r.bufferPool.Get()
	defer r.bufferPool.Put(b)

	result, err := r.render(req, info, b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, http.StatusInternalServerError)

		return
	}

	if r.config.Cache {
		body := make([]byte, b.Len())
		copy(body, b.Bytes())

		r.cache.Set(req.URL.Path, &indexRender{
			Body:           body,
			Status:         result.Status,
			Redirect:       result.Redirect,
			RedirectURL:    result.RedirectURL,
			RedirectStatus: result.RedirectStatus,
			Headers:        result.Headers,
		}, time.Duration(r.config.CacheTTL)*time.Second)
	}

	if result.Redirect {
		http.Redirect(w, req, result.RedirectURL, result.RedirectStatus)

		r.logger.Printf("Render completed (url=%s, redirect=%s, status=%d, cache=%t)", req.URL.Path, result.RedirectURL,
			result.RedirectStatus, false)

		return
	}

	w.WriteHeader(result.Status)
	w.Write(b.Bytes())

	r.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", req.URL.Path, result.Status, false)
}

// Next configures the next renderer
func (r *indexRenderer) Next(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *indexRenderer) render(req *http.Request, info *ServerInfo, w io.Writer) (*indexRender, error) {
	var valid bool = true
	var mServerState map[string]indexResourceResult
	var serverState *string
	var mClientState map[string]indexResourceResult
	var clientState *string
	var vmResult *vmResult
	if r.config.Bundle != nil {
		for index, rule := range r.config.Rules {
			if DEBUG {
				r.logger.Printf("Index: id=%s, rule=%d, phase=check, path=%s",
					req.Context().Value(ServerHandlerContextKeyRequestID{}).(string),
					index+1, req.URL.Path)
			}

			m := r.regexps[index].FindStringSubmatch(req.URL.Path)
			if m == nil {
				continue
			}

			if DEBUG {
				r.logger.Printf("Index: id=%s, rule=%d, phase=match, path=%s",
					req.Context().Value(ServerHandlerContextKeyRequestID{}).(string),
					index+1, req.URL.Path)
			}

			var params map[string]string
			if len(m) > 1 {
				params = make(map[string]string)
				for i, value := range m {
					if i > 0 {
						params[fmt.Sprint(i)] = value
					}
				}
				for i, name := range r.regexps[index].SubexpNames() {
					if i != 0 && name != "" {
						params[name] = m[i]
					}
				}
			}

			if DEBUG {
				r.logger.Printf("Index: id=%s, rule=%d, phase=params, path=%s, params=%s",
					req.Context().Value(ServerHandlerContextKeyRequestID{}).(string), index+1, req.URL.Path, params)
			}

			for _, entry := range rule.State {
				if mServerState == nil {
					mServerState = make(map[string]indexResourceResult)
				}
				if mClientState == nil && entry.Export != nil && *entry.Export {
					mClientState = make(map[string]indexResourceResult)
				}

				if DEBUG {
					r.logger.Printf("Index: id=%s, rule=%d, phase=state1, path=%s, state_key=%s, state_resource=%s",
						req.Context().Value(ServerHandlerContextKeyRequestID{}).(string), index+1, req.URL.Path, entry.Key,
						entry.Resource)
				}

				stateKey := r.replaceIndexRouteParameters(entry.Key, params)
				resourceKey := r.replaceIndexRouteParameters(entry.Resource, params)

				if DEBUG {
					r.logger.Printf("Index: id=%s, rule=%d, phase=state2, path=%s, state_key=%s, state_resource=%s",
						req.Context().Value(ServerHandlerContextKeyRequestID{}).(string), index+1, req.URL.Path, stateKey,
						resourceKey)
				}

				var resourceResult indexResourceResult
				response, err := r.fetcher.Get(resourceKey)
				if err != nil {
					if r.fetcher.Exists(resourceKey) {
						resourceResult.Loading = true
					} else {
						resourceResult.Error = indexResourceNotExist
					}

					mServerState[stateKey] = resourceResult
					if entry.Export != nil && *entry.Export {
						mClientState[stateKey] = resourceResult
					}

					valid = false

					continue
				}

				resourceResult.Response = string(response)

				mServerState[stateKey] = resourceResult
				if entry.Export != nil && *entry.Export {
					mClientState[stateKey] = resourceResult
				}
			}

			if mServerState != nil {
				buf, err := r.jsonMarshal(mServerState)
				if err != nil {
					r.logger.Printf("Failed to marshal server state: %s", err)

					return nil, err
				}

				s := string(buf)
				serverState = &s
			}

			if mClientState != nil {
				buf, err := r.jsonMarshal(mClientState)
				if err != nil {
					r.logger.Printf("Failed to marshal client state: %s", err)

					return nil, err
				}

				s := string(buf)
				clientState = &s
			}

			if r.config.Rules[index].Last {
				if DEBUG {
					r.logger.Printf("Index: id=%s, rule=%d, phase=last, path=%s",
						req.Context().Value(ServerHandlerContextKeyRequestID{}).(string),
						index+1, req.URL.Path)
				}

				break
			}
		}

		if DEBUG {
			if serverState != nil {
				r.logger.Printf("Index: id=%s, path=%s, server_state=%s",
					req.Context().Value(ServerHandlerContextKeyRequestID{}).(string),
					req.URL.Path, *serverState)
			}
			if clientState != nil {
				r.logger.Printf("Index: id=%s, path=%s, client_state=%s",
					req.Context().Value(ServerHandlerContextKeyRequestID{}).(string),
					req.URL.Path, *clientState)
			}
		}

		bundleInfo, err := r.osStat(*r.config.Bundle)
		if err != nil {
			r.logger.Printf("Failed to stat bundle file '%s': %s", *r.config.Bundle, err)

			return nil, err
		}
		if r.bundle == nil || r.bundleInfo == nil || bundleInfo.ModTime().After(*r.bundleInfo) {
			buf, err := r.osReadFile(*r.config.Bundle)
			if err != nil {
				r.logger.Printf("Failed to read bundle file '%s': %s", *r.config.Bundle, err)

				return nil, err
			}
			b := string(buf)
			r.bundle = &b
			i := bundleInfo.ModTime()
			r.bundleInfo = &i
		}

		var vm = r.vmPool.Get()
		defer r.vmPool.Put(vm)

		err = vm.Configure(r.config.Env, info, req, serverState)
		if err != nil {
			r.logger.Printf("Failed to configure VM: %s", err)

			return nil, err
		}

		vmResult, err = vm.Execute(*r.config.Bundle, *r.bundle, time.Duration(r.config.Timeout)*time.Second)
		if err != nil {
			r.logger.Printf("Failed to execute VM: %s", err)

			return nil, err
		}

		if vmResult.Redirect != nil && *vmResult.Redirect && vmResult.RedirectURL != nil && vmResult.RedirectStatus != nil {
			return &indexRender{
				Redirect:       *vmResult.Redirect,
				RedirectURL:    *vmResult.RedirectURL,
				RedirectStatus: *vmResult.RedirectStatus,
				Headers:        vmResult.Headers,
			}, nil
		}
	}

	htmlInfo, err := r.osStat(r.config.HTML)
	if err != nil {
		r.logger.Printf("Failed to stat html file '%s': %s", r.config.HTML, err)

		return nil, err
	}
	if r.html == nil || r.htmlInfo == nil || htmlInfo.ModTime().After(*r.htmlInfo) {
		buf, err := r.osReadFile(r.config.HTML)
		if err != nil {
			r.logger.Printf("Failed to read HTML file '%s': %s", r.config.HTML, err)

			return nil, err
		}

		r.html = &buf
		i := htmlInfo.ModTime()
		r.htmlInfo = &i
	}

	page := indexPage{
		HTML: r.html,
	}
	if vmResult != nil {
		page.Render = vmResult.Render
		page.State = clientState
		page.Title = vmResult.Title
		page.Metas = vmResult.Metas
		page.Links = vmResult.Links
		page.Scripts = vmResult.Scripts
	}

	err = r.index(&page, req, w)
	if err != nil {
		r.logger.Printf("Failed to render: %s", err)

		return nil, err
	}

	var status int = http.StatusOK
	if !valid {
		status = http.StatusServiceUnavailable
	}

	indexRender := indexRender{
		Status: status,
	}
	if vmResult != nil {
		if vmResult.Status != nil {
			indexRender.Status = *vmResult.Status
		}
		if vmResult.Redirect != nil {
			indexRender.Redirect = *vmResult.Redirect
		}
		if vmResult.RedirectURL != nil {
			indexRender.RedirectURL = *vmResult.RedirectURL
		}
		if vmResult.RedirectStatus != nil {
			indexRender.RedirectStatus = *vmResult.RedirectStatus
		}
		indexRender.Headers = vmResult.Headers
	}

	return &indexRender, nil
}

// index generates the final index response body
func (r *indexRenderer) index(page *indexPage, req *http.Request, w io.Writer) error {
	var body []byte

	body = *page.HTML

	if page.Render != nil {
		var old strings.Builder
		old.WriteString("<div id=\"")
		old.WriteString(r.config.Container)
		old.WriteString("\"></div>")

		var new strings.Builder
		new.WriteString("<div id=\"")
		new.WriteString(r.config.Container)
		new.WriteString("\">")
		new.Write(*page.Render)
		new.WriteString("</div>")

		body = bytes.Replace(body, []byte(old.String()), []byte(new.String()), 1)
	}

	if page.State != nil {
		var b strings.Builder
		b.WriteString("<script id=\"")
		b.WriteString(r.config.State)
		b.WriteString("\" type=\"application/json\">")
		b.WriteString(*page.State)
		b.WriteString("</script></body>")

		body = bytes.Replace(body, []byte("</body>"), []byte(b.String()), 1)
	}

	if page.Title != nil && *page.Title != "" {
		var b strings.Builder
		b.WriteString("<title>")
		b.WriteString(*page.Title)
		b.WriteString("</title></head>")

		body = bytes.Replace(body, []byte("</head>"), []byte(b.String()), 1)
	}

	if page.Metas != nil {
		for _, id := range page.Metas.Ids() {
			e, err := page.Metas.Get(id)
			if err != nil {
				continue
			}

			var b strings.Builder
			b.WriteString("<meta id=\"")
			b.WriteString(id)
			b.WriteString("\"")
			for _, k := range e.Attributes() {
				b.WriteString(" ")
				b.WriteString(k)
				b.WriteString("=\"")
				b.WriteString(e.GetAttribute(k))
				b.WriteString("\"")
			}
			b.WriteString("></head>")

			body = bytes.Replace(body, []byte("</head>"), []byte(b.String()), 1)
		}
	}

	if page.Links != nil {
		for _, id := range page.Links.Ids() {
			e, err := page.Links.Get(id)
			if err != nil {
				continue
			}

			var b strings.Builder
			b.WriteString("<link id=\"")
			b.WriteString(id)
			b.WriteString("\"")
			for _, k := range e.Attributes() {
				b.WriteString(" ")
				b.WriteString(k)
				b.WriteString("=\"")
				b.WriteString(e.GetAttribute(k))
				b.WriteString("\"")
			}
			b.WriteString("></head>")

			body = bytes.Replace(body, []byte("</head>"), []byte(b.String()), 1)
		}
	}

	if page.Scripts != nil {
		for _, id := range page.Scripts.Ids() {
			e, err := page.Scripts.Get(id)
			if err != nil {
				continue
			}

			var b strings.Builder
			b.WriteString("<script id=\"")
			b.WriteString(id)
			b.WriteString("\"")
			for _, k := range e.Attributes() {
				if k == "children" {
					continue
				}
				b.WriteString(" ")
				b.WriteString(k)
				b.WriteString("=\"")
				b.WriteString(e.GetAttribute(k))
				b.WriteString("\"")
			}
			b.WriteString(">")
			children := e.GetAttribute("children")
			if children != "" {
				b.WriteString(children)
			}
			b.WriteString("</script></head>")

			body = bytes.Replace(body, []byte("</head>"), []byte(b.String()), 1)
		}
	}

	w.Write(body)

	return nil
}

// replaceIndexRouteParameters returns a copy of the string s with all its parameters replaced
func (r *indexRenderer) replaceIndexRouteParameters(s string, params map[string]string) string {
	tmp := s
	for key, value := range params {
		tmp = strings.ReplaceAll(tmp, fmt.Sprint("$", key), value)
	}
	return tmp
}
