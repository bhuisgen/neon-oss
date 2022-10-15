// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
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
		vmPool:      newVMPool(int32(runtime.NumCPU())),
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
	result, err := r.render(req, info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})

		r.logger.Printf("Render error (url=%s, status=%d)", req.URL.Path, http.StatusInternalServerError)

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

// Next configures the next renderer
func (r *indexRenderer) Next(renderer Renderer) {
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
	var mServerState map[string]indexResourceResult
	var serverState *string
	var mClientState map[string]indexResourceResult
	var clientState *string
	var vmResult *vmResult
	if r.config.Bundle != nil {
		for index, rule := range r.config.Rules {
			if DEBUG {
				r.logger.Printf("Index: id=%s, rule=%d, phase=check, path=%s", req.Context().Value(ContextKeyID{}).(string),
					index+1, req.URL.Path)
			}

			m := r.regexps[index].FindStringSubmatch(req.URL.Path)
			if m == nil {
				continue
			}

			if DEBUG {
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

			if DEBUG {
				r.logger.Printf("Index: id=%s, rule=%d, phase=params, path=%s, params=%s",
					req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, params)
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
						req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, entry.Key, entry.Resource)
				}

				stateKey := replaceIndexRouteParameters(entry.Key, params)
				resourceKey := replaceIndexRouteParameters(entry.Resource, params)

				if DEBUG {
					r.logger.Printf("Index: id=%s, rule=%d, phase=state2, path=%s, state_key=%s, state_resource=%s",
						req.Context().Value(ContextKeyID{}).(string), index+1, req.URL.Path, stateKey, resourceKey)
				}

				var resourceResult indexResourceResult
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
					r.logger.Printf("Index: id=%s, rule=%d, phase=last, path=%s", req.Context().Value(ContextKeyID{}).(string),
						index+1, req.URL.Path)
				}

				break
			}
		}

		if DEBUG {
			if serverState != nil {
				r.logger.Printf("Index: id=%s, path=%s, server_state=%s", req.Context().Value(ContextKeyID{}).(string),
					req.URL.Path, *serverState)
			}
			if clientState != nil {
				r.logger.Printf("Index: id=%s, path=%s, client_state=%s", req.Context().Value(ContextKeyID{}).(string),
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

		if vmResult.Redirect {
			return &Render{
				Redirect:       true,
				RedirectTarget: vmResult.RedirectURL,
				RedirectStatus: vmResult.RedirectStatus,
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
	if vmResult != nil {
		result.Headers = vmResult.Headers
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

	if page.Title != nil && *page.Title != "" {
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<title>%s</title></head>", *page.Title)),
			1)
	}

	for id, attributes := range page.Metas {
		buf := r.bufferPool.Get()
		defer r.bufferPool.Put(buf)

		for k, v := range attributes {
			buf.WriteString(fmt.Sprintf(" %s=\"%s\"", k, v))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<meta id=\"%s\" name=\"%s\"%s/></head>", id, id, buf.String())),
			1)
	}

	for id, attributes := range page.Links {
		buf := r.bufferPool.Get()
		defer r.bufferPool.Put(buf)

		for k, v := range attributes {
			buf.WriteString(fmt.Sprintf(" %s=\"%s\"", k, v))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<link id=\"%s\"%s/></head>", id, buf.String())),
			1)
	}

	for id, attributes := range page.Scripts {
		buf := r.bufferPool.Get()
		defer r.bufferPool.Put(buf)

		var content string = ""
		for k, v := range attributes {
			if k == "children" {
				content = v
				continue
			}
			buf.WriteString(fmt.Sprintf(" %s=\"%s\"", k, v))
		}
		body = bytes.Replace(body,
			[]byte("</head>"),
			[]byte(fmt.Sprintf("<script id=\"%s\"%s>%s</script></head>", id, buf.String(), content)),
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
