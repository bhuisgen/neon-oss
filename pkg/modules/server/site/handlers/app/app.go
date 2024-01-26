// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/html"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// appHandler implements the app handler.
type appHandler struct {
	config      *appHandlerConfig
	logger      *log.Logger
	regexps     []*regexp.Regexp
	index       []byte
	indexInfo   *time.Time
	muIndex     *sync.RWMutex
	bundle      string
	bundleInfo  *time.Time
	muBundle    *sync.RWMutex
	rwPool      render.RenderWriterPool
	vmPool      VMPool
	cache       Cache
	site        core.ServerSite
	osOpen      func(name string) (*os.File, error)
	osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osReadFile  func(name string) ([]byte, error)
	osClose     func(*os.File) error
	osStat      func(name string) (fs.FileInfo, error)
	jsonMarshal func(v any) ([]byte, error)
}

// appHandlerConfig implements the app handler configuration.
type appHandlerConfig struct {
	Index         string
	Bundle        string
	Env           *string
	Container     *string
	State         *string
	Timeout       *int
	MaxVMs        *int
	Cache         *bool
	CacheTTL      *int
	CacheMaxItems *int
	Rules         []AppRule
}

// AppRule implements a rule.
type AppRule struct {
	Path  string
	State []AppRuleStateEntry
	Last  bool
}

// AppRuleStateEntry implements a rule state entry.
type AppRuleStateEntry struct {
	Key      string
	Resource string
	Export   *bool
}

// appCacheItem implements a app handler cache item.
type appCacheItem struct {
	render render.Render
	expire time.Time
}

// appResource implements a resource.
type appResource struct {
	Data  []string `json:"data"`
	Error string   `json:"error"`
}

const (
	appModuleID module.ModuleID = "server.site.handler.app"

	appResourceUnknown string = "unknown resource"

	appConfigDefaultBundleCodeCache bool   = false
	appConfigDefaultEnv             string = "production"
	appConfigDefaultContainer       string = "root"
	appConfigDefaultState           string = "state"
	appConfigDefaultTimeout         int    = 4
	appConfigDefaultCache           bool   = false
	appConfigDefaultCacheTTL        int    = 60
	appConfigDefaultCacheMaxItems   int    = 100
)

// appOsOpen redirects to os.Open.
func appOsOpen(name string) (*os.File, error) {
	return os.Open(name)
}

// appOsOpenFile redirects to os.OpenFile.
func appOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// appOsReadFile redirects to os.ReadFile.
func appOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// appOsClose redirects to os.Close.
func appOsClose(f *os.File) error {
	return f.Close()
}

// appOsStat redirects to os.Stat.
func appOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// appJsonMarshal redirects to json.Marshal.
func appJsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// init initializes the module.
func init() {
	module.Register(appHandler{})
}

// ModuleInfo returns the module information.
func (h appHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: appModuleID,
		NewInstance: func() module.Module {
			return &appHandler{
				muIndex:     new(sync.RWMutex),
				muBundle:    new(sync.RWMutex),
				osOpen:      appOsOpen,
				osOpenFile:  appOsOpenFile,
				osReadFile:  appOsReadFile,
				osClose:     appOsClose,
				osStat:      appOsStat,
				jsonMarshal: appJsonMarshal,
			}
		},
	}
}

// Init initializes the handler.
func (h *appHandler) Init(config map[string]interface{}, logger *log.Logger) error {
	h.logger = logger

	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Print("failed to parse configuration")
		return err
	}

	var errInit bool

	if h.config.Index == "" {
		h.logger.Printf("option '%s', missing option or value", "Index")
		errInit = true
	} else {
		f, err := h.osOpenFile(h.config.Index, os.O_RDONLY, 0)
		if err != nil {
			h.logger.Printf("option '%s', failed to open file '%s'", "Index", h.config.Index)
			errInit = true
		} else {
			h.osClose(f)
			fi, err := h.osStat(h.config.Index)
			if err != nil {
				h.logger.Printf("option '%s', failed to stat file '%s'", "Index", h.config.Index)
				errInit = true
			}
			if err == nil && fi.IsDir() {
				h.logger.Printf("option '%s', '%s' is a directory", "Index", h.config.Index)
				errInit = true
			}
		}
	}
	if h.config.Bundle == "" {
		h.logger.Printf("option '%s', missing option or value", "Bundle")
		errInit = true
	} else {
		f, err := h.osOpenFile(h.config.Bundle, os.O_RDONLY, 0)
		if err != nil {
			h.logger.Printf("option '%s', failed to open file '%s'", "Bundle", h.config.Bundle)
			errInit = true
		} else {
			h.osClose(f)
			fi, err := h.osStat(h.config.Bundle)
			if err != nil {
				h.logger.Printf("option '%s', failed to stat file '%s'", "Bundle", h.config.Bundle)
				errInit = true
			}
			if err == nil && fi.IsDir() {
				h.logger.Printf("option '%s', '%s' is a directory", "Bundle", h.config.Bundle)
				errInit = true
			}
		}
	}
	if h.config.Env == nil {
		defaultValue := appConfigDefaultEnv
		h.config.Env = &defaultValue
	}
	if *h.config.Env == "" {
		h.logger.Printf("option '%s', invalid value '%s'", "Env", *h.config.Env)
		errInit = true
	}
	if h.config.Container == nil {
		defaultValue := appConfigDefaultContainer
		h.config.Container = &defaultValue
	}
	if *h.config.Container == "" {
		h.logger.Printf("option '%s', invalid value '%s'", "Container", *h.config.Container)
		errInit = true
	}
	if h.config.State == nil {
		defaultValue := appConfigDefaultState
		h.config.State = &defaultValue
	}
	if *h.config.State == "" {
		h.logger.Printf("option '%s', invalid value '%s'", "State", *h.config.State)
		errInit = true
	}
	if h.config.Timeout == nil {
		defaultValue := appConfigDefaultTimeout
		h.config.Timeout = &defaultValue
	}
	if *h.config.Timeout < 0 {
		h.logger.Printf("option '%s', invalid value '%d'", "Timeout", *h.config.Timeout)
		errInit = true
	}
	if h.config.MaxVMs == nil {
		defaultValue := runtime.GOMAXPROCS(0)
		h.config.MaxVMs = &defaultValue
	}
	if *h.config.MaxVMs < 0 {
		h.logger.Printf("option '%s', invalid value '%d'", "MaxVMs", *h.config.MaxVMs)
		errInit = true
	}
	if h.config.Cache == nil {
		defaultValue := appConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := appConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}
	if *h.config.CacheTTL < 0 {
		h.logger.Printf("option '%s', invalid value '%d'", "CacheTTL", *h.config.CacheTTL)
		errInit = true
	}
	if h.config.CacheMaxItems == nil {
		defaultValue := appConfigDefaultCacheMaxItems
		h.config.CacheMaxItems = &defaultValue
	}
	if *h.config.CacheMaxItems < 0 {
		h.logger.Printf("option '%s', invalid value '%d'", "CacheMaxCapacity", *h.config.CacheMaxItems)
		errInit = true
	}
	for _, rule := range h.config.Rules {
		if rule.Path == "" {
			h.logger.Printf("rule option '%s', missing option or value", "Path")
			errInit = true
		} else {
			re, err := regexp.Compile(rule.Path)
			if err != nil {
				h.logger.Printf("rule option '%s', invalid regular expression '%s'", "Path", rule.Path)
				errInit = true
			} else {
				h.regexps = append(h.regexps, re)
			}
		}
		for _, state := range rule.State {
			if state.Key == "" {
				h.logger.Printf("rule state option '%s', missing option or value", "Key")
				errInit = true
			}
			if state.Resource == "" {
				h.logger.Printf("rule state option '%s', missing option or value", "Resource")
				errInit = true
			}
		}
	}

	if errInit {
		return errors.New("init error")
	}

	h.rwPool = render.NewRenderWriterPool()
	h.vmPool = newVMPool(*h.config.MaxVMs)
	h.cache = newCache(*h.config.CacheMaxItems)

	return nil
}

// Register registers the handler.
func (h *appHandler) Register(site core.ServerSite) error {
	h.site = site

	err := site.RegisterHandler(h)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the handler.
func (h *appHandler) Start() error {
	err := h.read()
	if err != nil {
		return err
	}

	return nil
}

// Stop stops the handler.
func (h *appHandler) Stop() {
	h.muIndex.Lock()
	h.indexInfo = nil
	h.muIndex.Unlock()

	h.muBundle.Lock()
	h.bundleInfo = nil
	h.muBundle.Unlock()

	h.cache.Clear()
}

// ServeHTTP implements the http handler.
func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path

	if *h.config.Cache {
		if item, ok := h.cache.Get(key).(appCacheItem); ok && item.expire.After(time.Now()) {
			render := item.render

			if render.Redirect() {
				http.Redirect(w, r, render.RedirectURL(), render.StatusCode())

				h.logger.Printf("Render completed (url=%s, redirect=%s, status=%d, cache=%t)", r.URL.Path,
					render.RedirectURL(), render.StatusCode(), true)

				return
			}

			for key, values := range render.Header() {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(render.StatusCode())
			if _, err := w.Write(render.Body()); err != nil {
				h.logger.Printf("Failed to write render: %s", err)

				return
			}

			h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, render.StatusCode(), true)

			return
		}
	}

	err := h.read()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Printf("Render error (url=%s, status=%d)", r.URL.Path, http.StatusServiceUnavailable)

		return
	}

	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	err = h.render(rw, r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Printf("Render error (url=%s, status=%d)", r.URL.Path, http.StatusServiceUnavailable)

		return
	}

	render := rw.Render()

	if *h.config.Cache {
		h.cache.Set(key, &appCacheItem{
			render: render,
			expire: time.Now().Add(time.Duration(*h.config.CacheTTL) * time.Second),
		})
	}

	if render.Redirect() {
		http.Redirect(w, r, render.RedirectURL(), render.StatusCode())

		h.logger.Printf("Render completed (url=%s, redirect=%s, status=%d, cache=%t)", r.URL.Path, render.RedirectURL(),
			render.StatusCode(), false)

		return
	}

	for key, values := range rw.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(render.StatusCode())
	if _, err := w.Write(render.Body()); err != nil {
		h.logger.Printf("Failed to write render: %s", err)

		return
	}

	h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, rw.StatusCode(), false)
}

// read reads the application html and bundle files.
func (h *appHandler) read() error {
	htmlInfo, err := h.osStat(h.config.Index)
	if err != nil {
		h.logger.Printf("Failed to stat index file '%s': %s", h.config.Index, err)

		return err
	}

	h.muIndex.RLock()
	if h.indexInfo == nil || htmlInfo.ModTime().After(*h.indexInfo) {
		h.muIndex.RUnlock()

		buf, err := h.osReadFile(h.config.Index)
		if err != nil {
			h.logger.Printf("Failed to read index file '%s': %s", h.config.Index, err)

			return err
		}

		h.muIndex.Lock()
		h.index = buf
		i := htmlInfo.ModTime()
		h.indexInfo = &i
		h.muIndex.Unlock()
	} else {
		h.muIndex.RUnlock()
	}

	bundleInfo, err := h.osStat(h.config.Bundle)
	if err != nil {
		h.logger.Printf("Failed to stat bundle file '%s': %s", h.config.Bundle, err)

		return err
	}

	h.muBundle.RLock()
	if h.bundleInfo == nil || bundleInfo.ModTime().After(*h.bundleInfo) {
		h.muBundle.RUnlock()

		buf, err := h.osReadFile(h.config.Bundle)
		if err != nil {
			h.logger.Printf("Failed to read bundle file '%s': %s", h.config.Bundle, err)

			return err
		}

		h.muBundle.Lock()
		h.bundle = string(buf)
		i := bundleInfo.ModTime()
		h.bundleInfo = &i
		h.muBundle.Unlock()
	} else {
		h.muBundle.RUnlock()
	}

	return nil
}

// render makes a new render.
func (h *appHandler) render(w render.RenderWriter, r *http.Request) error {
	var valid bool = true
	var mServerState map[string]appResource
	var serverState *string
	var mClientState map[string]appResource
	var clientState *string
	var vmResult *vmResult

	for index, rule := range h.config.Rules {
		if debug, ok := os.LookupEnv("DEBUG"); ok && debug == "1" {
			h.logger.Printf("rule=%d, phase=check, path=%s", index+1, r.URL.Path)
		}

		m := h.regexps[index].FindStringSubmatch(r.URL.Path)
		if m == nil {
			continue
		}

		if debug, ok := os.LookupEnv("DEBUG"); ok && debug == "1" {
			h.logger.Printf("rule=%d, phase=match, path=%s", index+1, r.URL.Path)
		}

		params := make(map[string]string)
		params["url"] = r.URL.Path
		if len(m) > 1 {
			for i, value := range m {
				if i > 0 {
					params[fmt.Sprint(i)] = value
				}
			}
			for i, name := range h.regexps[index].SubexpNames() {
				if i != 0 && name != "" {
					params[name] = m[i]
				}
			}
		}

		if debug, ok := os.LookupEnv("DEBUG"); ok && debug == "1" {
			h.logger.Printf("rule=%d, phase=params, path=%s, params=%s", index+1, r.URL.Path, params)
		}

		for _, entry := range rule.State {
			if mServerState == nil {
				mServerState = make(map[string]appResource)
			}
			if mClientState == nil && entry.Export != nil && *entry.Export {
				mClientState = make(map[string]appResource)
			}

			if debug, ok := os.LookupEnv("DEBUG"); ok && debug == "1" {
				h.logger.Printf("rule=%d, phase=state1, path=%s, state_key=%s, state_resource=%s", index+1, r.URL.Path,
					entry.Key, entry.Resource)
			}

			stateKey := h.replaceIndexRouteParameters(entry.Key, params)
			resourceKey := h.replaceIndexRouteParameters(entry.Resource, params)

			if debug, ok := os.LookupEnv("DEBUG"); ok && debug == "1" {
				h.logger.Printf("rule=%d, phase=state2, path=%s, state_key=%s, state_resource=%s", index+1, r.URL.Path,
					stateKey, resourceKey)
			}

			var resourceResult appResource
			resource, err := h.site.Store().LoadResource(resourceKey)
			if err != nil {
				resourceResult.Error = appResourceUnknown

				mServerState[stateKey] = resourceResult
				if entry.Export != nil && *entry.Export {
					mClientState[stateKey] = resourceResult
				}

				valid = false

				continue
			}

			resourceResult.Data = make([]string, len(resource.Data))
			for index := range resource.Data {
				resourceResult.Data[index] = string(resource.Data[index])
			}

			mServerState[stateKey] = resourceResult
			if entry.Export != nil && *entry.Export {
				mClientState[stateKey] = resourceResult
			}
		}

		if mServerState != nil {
			buf, err := h.jsonMarshal(mServerState)
			if err != nil {
				h.logger.Printf("Failed to marshal server state: %s", err)

				return err
			}

			s := string(buf)
			serverState = &s
		}

		if mClientState != nil {
			buf, err := h.jsonMarshal(mClientState)
			if err != nil {
				h.logger.Printf("Failed to marshal client state: %s", err)

				return err
			}

			s := string(buf)
			clientState = &s
		}

		if h.config.Rules[index].Last {
			if debug, ok := os.LookupEnv("DEBUG"); ok && debug == "1" {
				h.logger.Printf("rule=%d, phase=last, path=%s", index+1, r.URL.Path)
			}

			break
		}
	}

	if debug, ok := os.LookupEnv("DEBUG"); ok && debug == "1" {
		if serverState != nil {
			h.logger.Printf("path=%s, server_state=%s", r.URL.Path, *serverState)
		}
		if clientState != nil {
			h.logger.Printf("path=%s, client_state=%s", r.URL.Path, *clientState)
		}
	}

	var vm = h.vmPool.Get()
	defer h.vmPool.Put(vm)

	if err := vm.Configure(&vmConfig{
		Env:     *h.config.Env,
		Request: r,
		State:   serverState,
	}); err != nil {
		h.logger.Printf("Failed to configure VM: %s", err)

		return err
	}

	var err error
	h.muBundle.RLock()
	result, err := vm.Execute(h.config.Bundle, h.bundle, time.Duration(*h.config.Timeout)*time.Second)
	h.muBundle.RUnlock()
	if err != nil {
		h.logger.Printf("Failed to execute VM: %s", err)

		return err
	}

	vmResult = result

	if vmResult.Redirect != nil && *vmResult.Redirect && vmResult.RedirectURL != nil && vmResult.RedirectStatus != nil {
		w.WriteRedirect(*vmResult.RedirectURL, *vmResult.RedirectStatus)

		return nil
	}

	for key, value := range vmResult.Headers {
		for _, v := range value {
			w.Header().Add(key, v)
		}
	}
	if valid {
		w.WriteHeader(*vmResult.Status)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	h.muIndex.RLock()
	if h.index != nil {
		err = h.app(w, r, bytes.NewReader(h.index), clientState, vmResult)
	} else {
		err = errors.New("index not loaded")
	}
	h.muIndex.RUnlock()
	if err != nil {
		h.logger.Printf("Failed to render: %s", err)

		return err
	}

	return nil
}

// app writes the final index.
func (h *appHandler) app(w render.RenderWriter, r *http.Request, b io.Reader, state *string, result *vmResult) error {
	doc, err := html.Parse(b)
	if err != nil {
		return err
	}

	if result.Render != nil {
		var renderContainer func(*html.Node) bool
		renderContainer = func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "div" {
				for _, d := range n.Attr {
					if d.Key == "id" && d.Val == *h.config.Container {
						n.AppendChild(&html.Node{
							Type: html.RawNode,
							Data: string(*result.Render),
						})
						return true
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if renderContainer(c) {
					return true
				}
			}
			return false
		}
		if !renderContainer(doc) {
			return errors.New("container not found")
		}
	}

	if state != nil {
		var renderState func(*html.Node) bool
		renderState = func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "body" {
				n.AppendChild(&html.Node{
					Type: html.ElementNode,
					Data: "script",
					Attr: []html.Attribute{
						{
							Key: "id",
							Val: *h.config.State,
						},
						{
							Key: "type",
							Val: "application/json",
						},
					},
					FirstChild: &html.Node{
						Type: html.RawNode,
						Data: *state,
					},
				})
				return true
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if renderState(c) {
					return true
				}
			}
			return false
		}
		if !renderState(doc) {
			return errors.New("body not found")
		}
	}

	if result.Title != nil {
		var renderTitle func(*html.Node) bool
		renderTitle = func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "head" {
				n.AppendChild(&html.Node{
					Type: html.ElementNode,
					Data: "title",
					FirstChild: &html.Node{
						Type: html.TextNode,
						Data: *result.Title,
					},
				})
				return true
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if renderTitle(c) {
					return true
				}
			}
			return false
		}
		if !renderTitle(doc) {
			return errors.New("head not found")
		}
	}

	if result.Metas != nil {
		var renderMetas func(*html.Node) bool
		renderMetas = func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "head" {
				for _, id := range result.Metas.Ids() {
					e, err := result.Metas.Get(id)
					if err != nil {
						continue
					}
					var attrs []html.Attribute
					attrs = append(attrs, html.Attribute{
						Key: "id",
						Val: id,
					})
					for _, k := range e.Attributes() {
						attrs = append(attrs, html.Attribute{
							Key: k,
							Val: e.GetAttribute(k),
						})
					}
					n.AppendChild(&html.Node{
						Type: html.ElementNode,
						Data: "meta",
						Attr: attrs,
					})
				}
				return true
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if renderMetas(c) {
					return true
				}
			}
			return false
		}
		if !renderMetas(doc) {
			return errors.New("head not found")
		}
	}

	if result.Links != nil {
		var renderLink func(*html.Node) bool
		renderLink = func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "head" {
				for _, id := range result.Links.Ids() {
					e, err := result.Links.Get(id)
					if err != nil {
						continue
					}
					var attrs []html.Attribute
					attrs = append(attrs, html.Attribute{
						Key: "id",
						Val: id,
					})
					for _, k := range e.Attributes() {
						attrs = append(attrs, html.Attribute{
							Key: k,
							Val: e.GetAttribute(k),
						})
					}
					n.AppendChild(&html.Node{
						Type: html.ElementNode,
						Data: "link",
						Attr: attrs,
					})
				}
				return true
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if renderLink(c) {
					return true
				}
			}
			return false
		}
		if !renderLink(doc) {
			return errors.New("head not found")
		}
	}

	if result.Scripts != nil {
		var renderScript func(*html.Node) bool
		renderScript = func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "head" {
				for _, id := range result.Scripts.Ids() {
					e, err := result.Scripts.Get(id)
					if err != nil {
						continue
					}
					var attrs []html.Attribute
					attrs = append(attrs, html.Attribute{
						Key: "id",
						Val: id,
					})
					for _, k := range e.Attributes() {
						if k == "children" {
							continue
						}
						attrs = append(attrs, html.Attribute{
							Key: k,
							Val: e.GetAttribute(k),
						})
					}
					children := e.GetAttribute("children")
					n.AppendChild(&html.Node{
						Type: html.ElementNode,
						Data: "script",
						Attr: attrs,
						FirstChild: &html.Node{
							Type: html.RawNode,
							Data: children,
						},
					})
				}
				return true
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if renderScript(c) {
					return true
				}
			}
			return false
		}
		if !renderScript(doc) {
			return errors.New("head not found")
		}
	}

	err = html.Render(w, doc)
	if err != nil {
		return err
	}

	return nil
}

// replaceIndexRouteParameters returns a copy of the string s with all its parameters replaced.
func (h *appHandler) replaceIndexRouteParameters(s string, params map[string]string) string {
	tmp := s
	for key, value := range params {
		tmp = strings.ReplaceAll(tmp, fmt.Sprint("$", key), value)
	}
	return tmp
}

var _ core.ServerSiteMiddlewareModule = (*appHandler)(nil)
