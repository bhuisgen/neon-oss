// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Unauthorized copying of this file, via any medium is strictly prohibited.

package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/bhuisgen/neon/pkg/cache"
	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
	"github.com/mitchellh/mapstructure"
)

// appHandler implements the app handler
type appHandler struct {
	config      *appHandlerConfig
	logger      *log.Logger
	regexps     []*regexp.Regexp
	index       []byte
	indexInfo   *time.Time
	bundle      string
	bundleInfo  *time.Time
	rwPool      render.RenderWriterPool
	vmPool      VMPool
	cache       cache.Cache
	store       core.Store
	fetcher     core.Fetcher
	osOpen      func(name string) (*os.File, error)
	osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osReadFile  func(name string) ([]byte, error)
	osClose     func(*os.File) error
	osStat      func(name string) (fs.FileInfo, error)
	jsonMarshal func(v any) ([]byte, error)
}

// appHandlerConfig implements the app handler configuration.
type appHandlerConfig struct {
	Index     string
	Bundle    string
	Env       *string
	Container *string
	State     *string
	Timeout   *int
	MaxVMs    *int
	Cache     *bool
	CacheTTL  *int
	Rules     []AppRule
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

// appResource implements a resource.
type appResource struct {
	Data  []string `json:"data"`
	Error string   `json:"error"`
}

const (
	appModuleID module.ModuleID = "server.handler.app"
	appLogger   string          = "server.handler.app"

	appResourceUnknown string = "unknown resource"

	appConfigDefaultBundleCodeCache bool   = false
	appConfigDefaultEnv             string = "production"
	appConfigDefaultContainer       string = "root"
	appConfigDefaultState           string = "state"
	appConfigDefaultTimeout         int    = 4
	appConfigDefaultMinSpareVMs     int    = 0
	appConfigDefaultMaxSpareVMs     int    = 0
	appConfigDefaultCache           bool   = false
	appConfigDefaultCacheTTL        int    = 60
	appConfigDefaultRuleLast        bool   = false
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

// Check checks the handler configuration.
func (h *appHandler) Check(config map[string]interface{}) ([]string, error) {
	var report []string

	var c appHandlerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		report = append(report, "failed to parse configuration")
		return report, err
	}

	if c.Index == "" {
		report = append(report, fmt.Sprintf("option '%s', missing option or value", "Index"))
	} else {
		f, err := h.osOpenFile(c.Index, os.O_RDONLY, 0)
		if err != nil {
			report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "Index", c.Index))
		} else {
			h.osClose(f)
			fi, err := h.osStat(c.Index)
			if err != nil {
				report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "Index", c.Index))
			}
			if err == nil && fi.IsDir() {
				report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "Index", c.Index))
			}
		}
	}
	if c.Bundle == "" {
		report = append(report, fmt.Sprintf("option '%s', missing option or value", "Bundle"))
	} else {
		f, err := h.osOpenFile(c.Bundle, os.O_RDONLY, 0)
		if err != nil {
			report = append(report, fmt.Sprintf("option '%s', failed to open file '%s'", "Bundle", c.Bundle))
		} else {
			h.osClose(f)
			fi, err := h.osStat(c.Bundle)
			if err != nil {
				report = append(report, fmt.Sprintf("option '%s', failed to stat file '%s'", "Bundle", c.Bundle))
			}
			if err == nil && fi.IsDir() {
				report = append(report, fmt.Sprintf("option '%s', '%s' is a directory", "Bundle", c.Bundle))
			}
		}
	}
	if c.Env == nil {
		defaultValue := appConfigDefaultEnv
		c.Env = &defaultValue
	}
	if *c.Env == "" {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "Env", *c.Env))
	}
	if c.Container == nil {
		defaultValue := appConfigDefaultContainer
		c.Container = &defaultValue
	}
	if *c.Container == "" {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "Container", *c.Container))
	}
	if c.State == nil {
		defaultValue := appConfigDefaultState
		c.State = &defaultValue
	}
	if *c.State == "" {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%s'", "State", *c.State))
	}
	if c.Timeout == nil {
		defaultValue := appConfigDefaultTimeout
		c.Timeout = &defaultValue
	}
	if *c.Timeout < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "Timeout", *c.Timeout))
	}
	if c.MaxVMs == nil {
		defaultValue := runtime.NumCPU() * 2
		c.MaxVMs = &defaultValue
	}
	if *c.MaxVMs < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "MaxVMs", *c.MaxVMs))
	}
	if c.CacheTTL == nil {
		defaultValue := appConfigDefaultCacheTTL
		c.CacheTTL = &defaultValue
	}
	if *c.CacheTTL < 0 {
		report = append(report, fmt.Sprintf("option '%s', invalid value '%d'", "CacheTTL", *c.CacheTTL))
	}
	for _, rule := range c.Rules {
		if rule.Path == "" {
			report = append(report, fmt.Sprintf("rule option '%s', missing option or value", "Path"))
		}
		for _, state := range rule.State {
			if state.Key == "" {
				report = append(report, fmt.Sprintf("rule state option '%s', missing option or value", "Key"))
			}
			if state.Resource == "" {
				report = append(report, fmt.Sprintf("rule state option '%s', missing option or value", "Resource"))
			}
		}
	}

	if len(report) > 0 {
		return report, errors.New("check failure")
	}

	return nil, nil
}

// Load loads the handler.
func (h *appHandler) Load(config map[string]interface{}) error {
	var c appHandlerConfig
	err := mapstructure.Decode(config, &c)
	if err != nil {
		return err
	}

	h.config = &c
	h.logger = log.New(os.Stderr, fmt.Sprint(appLogger, ": "), log.LstdFlags|log.Lmsgprefix)

	if h.config.Env == nil {
		defaultValue := appConfigDefaultEnv
		h.config.Env = &defaultValue
	}
	if h.config.Container == nil {
		defaultValue := appConfigDefaultContainer
		h.config.Container = &defaultValue
	}
	if h.config.State == nil {
		defaultValue := appConfigDefaultState
		h.config.State = &defaultValue
	}
	if h.config.Timeout == nil {
		defaultValue := appConfigDefaultTimeout
		h.config.Timeout = &defaultValue
	}
	if h.config.MaxVMs == nil {
		defaultValue := runtime.NumCPU() * 2
		h.config.MaxVMs = &defaultValue
	}
	if h.config.Cache == nil {
		defaultValue := appConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := appConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}

	h.rwPool = render.NewRenderWriterPool()
	h.cache = cache.NewCache()

	return nil
}

// Register registers the server resources.
func (h *appHandler) Register(registry core.ServerRegistry) error {
	err := registry.RegisterHandler(h)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the handler.
func (h *appHandler) Start(store core.Store, fetcher core.Fetcher) error {
	h.store = store
	h.fetcher = fetcher

	for _, rule := range h.config.Rules {
		re, err := regexp.Compile(rule.Path)
		if err != nil {
			return err
		}
		h.regexps = append(h.regexps, re)
	}
	h.vmPool = newVMPool(int32(*h.config.MaxVMs))

	err := h.read()
	if err != nil {
		return err
	}

	return nil
}

// Mount mounts the handler.
func (h *appHandler) Mount() error {
	return nil
}

// Unmount unmounts the handler.
func (h *appHandler) Unmount() {
}

// Stop stops the handler.
func (h *appHandler) Stop() {
	h.regexps = []*regexp.Regexp{}
	h.indexInfo = nil
	h.bundleInfo = nil
	h.vmPool = nil
	h.cache.Clear()
}

// ServeHTTP implements the http handler.
func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var key string
	key = r.URL.Path

	if *h.config.Cache {
		obj := h.cache.Get(key)
		if obj != nil {
			render := obj.(render.Render)

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
			w.Write(render.Body())

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
		h.cache.Set(key, render, time.Duration(*h.config.CacheTTL)*time.Second)
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
	w.Write(render.Body())

	h.logger.Printf("Render completed (url=%s, status=%d, cache=%t)", r.URL.Path, rw.StatusCode(), false)
}

// read reads the application html and bundle files.
func (h *appHandler) read() error {
	htmlInfo, err := h.osStat(h.config.Index)
	if err != nil {
		h.logger.Printf("Failed to stat index file '%s': %s", h.config.Index, err)

		return err
	}
	if h.indexInfo == nil || htmlInfo.ModTime().After(*h.indexInfo) {
		buf, err := h.osReadFile(h.config.Index)
		if err != nil {
			h.logger.Printf("Failed to read index file '%s': %s", h.config.Index, err)

			return err
		}

		h.index = buf
		i := htmlInfo.ModTime()
		h.indexInfo = &i
	}

	bundleInfo, err := h.osStat(h.config.Bundle)
	if err != nil {
		h.logger.Printf("Failed to stat bundle file '%s': %s", h.config.Bundle, err)

		return err
	}
	if h.bundleInfo == nil || bundleInfo.ModTime().After(*h.bundleInfo) {
		buf, err := h.osReadFile(h.config.Bundle)
		if err != nil {
			h.logger.Printf("Failed to read bundle file '%s': %s", h.config.Bundle, err)

			return err
		}
		h.bundle = string(buf)
		i := bundleInfo.ModTime()
		h.bundleInfo = &i
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
			resource, err := h.store.Get(resourceKey)
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

	err := vm.Configure(&vmConfig{
		Env:     *h.config.Env,
		Request: r,
		State:   serverState,
	})
	if err != nil {
		h.logger.Printf("Failed to configure VM: %s", err)

		return err
	}

	result, err := vm.Execute(h.config.Bundle, h.bundle, time.Duration(*h.config.Timeout)*time.Second)
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

	err = h.app(w, r, clientState, vmResult)
	if err != nil {
		h.logger.Printf("Failed to render: %s", err)

		return err
	}

	return nil
}

// app generates the final index response body.
func (h *appHandler) app(w render.RenderWriter, r *http.Request, state *string,
	result *vmResult) error {
	doc, err := html.Parse(bytes.NewReader(h.index))
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

var _ core.ServerMiddlewareModule = (*appHandler)(nil)
