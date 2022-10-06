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
	"regexp"
	"strings"
	"time"
)

// indexRenderer implements the index renderer
type indexRenderer struct {
	Renderer
	next Renderer

	config  *IndexRendererConfig
	logger  *log.Logger
	regexps []*regexp.Regexp
	vmPool  *vmPool
	cache   *cache
	fetcher *fetcher
}

// IndexRendererConfig implements the index renderer configuration
type IndexRendererConfig struct {
	Enable    bool
	HTML      string
	Bundle    *string
	Env       string
	Container string
	State     string
	Timeout   int
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

// indexPage implements a page
type indexPage struct {
	HTML    *[]byte
	Render  *[]byte
	Title   *string
	Metas   map[string]map[string]string
	Links   map[string]map[string]string
	Scripts map[string]map[string]string
	State   *string
}

const (
	indexLogger string = "server[index]"
)

// CreateIndexRenderer creates a new index renderer
func CreateIndexRenderer(config *IndexRendererConfig, fetcher *fetcher) (*indexRenderer, error) {
	logger := log.New(os.Stderr, fmt.Sprint(indexLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	regexps := []*regexp.Regexp{}
	for _, rule := range config.Rules {
		r, err := regexp.Compile(rule.Path)
		if err != nil {
			return nil, err
		}

		regexps = append(regexps, r)
	}

	return &indexRenderer{
		config:  config,
		logger:  logger,
		regexps: regexps,
		vmPool:  NewVMPool(),
		cache:   NewCache(),
		fetcher: fetcher,
	}, nil
}

// handle implements the renderer handler
func (r *indexRenderer) handle(w http.ResponseWriter, req *http.Request, info *ServerInfo) {
	result, err := r.render(req, info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, result.Status)

		return
	}

	if result.Redirect {
		http.Redirect(w, req, result.RedirectTarget, result.RedirectStatus)

		r.logger.Printf("Redirect completed (url=%s, status=%d, target=%s)", req.URL.Path, result.RedirectStatus,
			result.RedirectTarget)

		return
	}

	w.WriteHeader(result.Status)
	w.Write(result.Body)

	r.logger.Printf("Render completed (url=%s, status=%d, valid=%t, cache=%t)", req.URL.Path, result.Status, result.Valid,
		result.Cache)
}

// setNext configures the next renderer
func (r *indexRenderer) setNext(renderer Renderer) {
	r.next = renderer
}

// render makes a new render
func (r *indexRenderer) render(req *http.Request, info *ServerInfo) (*Render, error) {
	if r.config.Cache {
		obj := r.cache.Get(req.URL.Path)
		if obj != nil {
			result := obj.(*Render)

			return result, nil
		}
	}

	var valid bool = true
	var mServerState map[string]ResourceResult
	var serverState *string
	var mClientState map[string]ResourceResult
	var clientState *string

	for index, rule := range r.config.Rules {
		if _, ok := os.LookupEnv("DEBUG"); ok {
			r.logger.Printf("Index: id=%s, rule=%d, phase=check, path=%s", req.Context().Value(ContextKeyID{}).(string),
				index+1, req.URL.Path)
		}

		m := r.regexps[index].FindStringSubmatch(req.URL.Path)
		if m == nil {
			continue
		}

		if _, ok := os.LookupEnv("DEBUG"); ok {
			r.logger.Printf("Index: id=%s, rule=%d, phase=match, path=%s", req.Context().Value(ContextKeyID{}).(string),
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

		if _, ok := os.LookupEnv("DEBUG"); ok {
			r.logger.Printf("Index: id=%s, rule=%d, phase=params, path=%s, params=%s",
				req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, params)
		}

		for _, entry := range rule.State {
			if mServerState == nil {
				mServerState = make(map[string]ResourceResult)
			}
			if mClientState == nil && entry.Export != nil && *entry.Export {
				mClientState = make(map[string]ResourceResult)
			}

			if _, ok := os.LookupEnv("DEBUG"); ok {
				r.logger.Printf("Index: id=%s, rule=%d, phase=state1, path=%s, state_key=%s, state_resource=%s",
					req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, entry.Key, entry.Resource)
			}

			stateKey := replaceIndexRouteParameters(entry.Key, params)
			resourceKey := replaceIndexRouteParameters(entry.Resource, params)

			if _, ok := os.LookupEnv("DEBUG"); ok {
				r.logger.Printf("Index: id=%s, rule=%d, phase=state2, path=%s, state_key=%s, state_resource=%s",
					req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, stateKey, resourceKey)
			}

			var resourceResult ResourceResult
			response, err := r.fetcher.Get(resourceKey)
			if err != nil {
				if r.fetcher.Exists(resourceKey) {
					resourceResult.Loading = true
				} else {
					resourceResult.Error = "unknown resource"
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
			buf, err := json.Marshal(mServerState)
			if err != nil {
				r.logger.Printf("Failed to marshal server state: %s", err)

				return nil, err
			}

			s := string(buf)
			serverState = &s
		}

		if mClientState != nil {
			buf, err := json.Marshal(mClientState)
			if err != nil {
				r.logger.Printf("Failed to marshal client state: %s", err)

				return nil, err
			}

			s := string(buf)
			clientState = &s
		}

		if r.config.Rules[index].Last {
			if _, ok := os.LookupEnv("DEBUG"); ok {
				r.logger.Printf("Index: id=%s, rule=%d, phase=last, path=%s", req.Context().Value(ContextKeyID{}).(string),
					index+1, req.URL.Path)
			}

			break
		}
	}

	if _, ok := os.LookupEnv("DEBUG"); ok {
		if serverState != nil {
			r.logger.Printf("Index: id=%s, path=%s, server_state=%s", req.Context().Value(ContextKeyID{}).(string),
				req.URL.Path, *serverState)
		}
		if clientState != nil {
			r.logger.Printf("Index: id=%s, path=%s, client_state=%s", req.Context().Value(ContextKeyID{}).(string),
				req.URL.Path, *clientState)
		}
	}

	var vmResult *vmResult
	if r.config.Bundle != nil {
		var bundle string
		buf, err := os.ReadFile(*r.config.Bundle)
		if err != nil {
			r.logger.Printf("Failed to read bundle file '%s': %s", *r.config.Bundle, err)

			return nil, err
		}
		bundle = string(buf)

		var vm = r.vmPool.Get()
		defer r.vmPool.Put(vm)
		vm.Reset()

		err = vm.Configure(r.config.Env, info, req, serverState)
		if err != nil {
			r.logger.Printf("Failed to configure VM: %s", err)
		}

		vmResult, err = vm.Execute(*r.config.Bundle, bundle, r.config.Timeout)
		if err != nil {
			r.logger.Printf("Failed to execute VM: %s", err)

			return nil, err
		}

		if vmResult.Redirect {
			return &Render{
				Redirect:       true,
				RedirectTarget: vmResult.RedirectURL,
				RedirectStatus: vmResult.RedirectStatus,
			}, nil
		}
	}

	html, err := os.ReadFile(r.config.HTML)
	if err != nil {
		r.logger.Printf("Failed to read HTML file '%s': %s", r.config.HTML, err)

		return nil, err
	}

	page := indexPage{
		HTML: &html,
	}
	if vmResult != nil {
		page.Render = &vmResult.Render
		page.Title = &vmResult.Title
		page.Metas = vmResult.Metas
		page.Links = vmResult.Links
		page.Scripts = vmResult.Scripts
		page.State = clientState
	}

	body, err := index(&page, r, req)
	if err != nil {
		r.logger.Printf("Failed to generate index: %s", err)

		return nil, err
	}

	var status int = http.StatusOK
	if vmResult != nil {
		status = vmResult.Status
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

// index generates the final index response body
func index(page *indexPage, r *indexRenderer, req *http.Request) ([]byte, error) {
	var body []byte

	body = *page.HTML

	if page.Title != nil && *page.Title != "" {
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<title>%s</title></head>", *page.Title)),
			1)
	}

	for id, attributes := range page.Metas {
		var buf = bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		for k, v := range attributes {
			buf.WriteString(fmt.Sprintf(" %s=\"%s\"", k, v))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<meta id=\"%s\" name=\"%s\"%s/></head>", id, id, buf.String())),
			1)
	}

	for id, attributes := range page.Links {
		var buf = bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		for k, v := range attributes {
			buf.WriteString(fmt.Sprintf(" %s=\"%s\"", k, v))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<link id=\"%s\"%s/></head>", id, buf.String())),
			1)
	}

	for id, attributes := range page.Scripts {
		var buf = bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		var content string = ""
		for k, v := range attributes {
			if k == "children" {
				content = v
			}
			buf.WriteString(fmt.Sprintf(" %s=\"%s\"", k, v))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<script id=\"%s\"%s>%s</script></head>", id, buf.String(), content)),
			1)
	}

	if page.Render != nil {
		body = bytes.Replace(body,
			[]byte(fmt.Sprintf("<div id=\"%s\"></div>", r.config.Container)),
			[]byte(fmt.Sprintf("<div id=\"%s\">%s</div>", r.config.Container, *page.Render)),
			1)
	}

	if page.State != nil {
		body = bytes.Replace(body,
			[]byte("</body>"),
			[]byte(fmt.Sprintf("<script id=\"%s\" type=\"application/json\">%s</script></body>", r.config.State, *page.State)),
			1)
	}

	return body, nil
}

// replaceIndexRouteParameters returns a copy of the string s with all its parameters replaced
func replaceIndexRouteParameters(s string, params map[string]string) string {
	tmp := s
	for key, value := range params {
		tmp = strings.ReplaceAll(tmp, fmt.Sprint("$", key), value)
	}
	return tmp
}
