package js

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bhuisgen/gomonkey"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/html"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// jsHandler implements the js handler.
type jsHandler struct {
	config      *jsHandlerConfig
	logger      *slog.Logger
	regexps     []*regexp.Regexp
	index       []byte
	indexInfo   *time.Time
	muIndex     *sync.RWMutex
	bundle      []byte
	bundleInfo  *time.Time
	muBundle    *sync.RWMutex
	vms         chan struct{}
	rwPool      render.RenderWriterPool
	cache       Cache
	site        core.ServerSite
	osOpen      func(name string) (*os.File, error)
	osOpenFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osReadFile  func(name string) ([]byte, error)
	osClose     func(*os.File) error
	osStat      func(name string) (fs.FileInfo, error)
	jsonMarshal func(v any) ([]byte, error)
}

// jsHandlerConfig implements the js handler configuration.
type jsHandlerConfig struct {
	Index         string   `mapstructure:"index"`
	Bundle        string   `mapstructure:"bundle"`
	Env           *string  `mapstructure:"env"`
	Container     *string  `mapstructure:"container"`
	State         *string  `mapstructure:"state"`
	MaxVMs        *int     `mapstructure:"maxVMs"`
	VMMaxHeapSize *int     `mapstructure:"vmMaxHeapSize"`
	VMStackSize   *int     `mapstructure:"vmStackSize"`
	VMTimeout     *int     `mapstructure:"vmTimeout"`
	Cache         *bool    `mapstructure:"cache"`
	CacheTTL      *int     `mapstructure:"cacheTTL"`
	CacheMaxItems *int     `mapstructure:"cacheMaxItems"`
	Rules         []JSRule `mapstructure:"rules"`
}

// JSRule implements a rule.
type JSRule struct {
	Path  string             `mapstructure:"path"`
	State []JSRuleStateEntry `mapstructure:"state"`
	Last  bool               `mapstructure:"last"`
}

// JSRuleStateEntry implements a rule state entry.
type JSRuleStateEntry struct {
	Key      string `mapstructure:"key"`
	Resource string `mapstructure:"resource"`
	Export   *bool  `mapstructure:"export"`
}

// jsCacheItem implements a cached item.
type jsCacheItem struct {
	render render.Render
	expire time.Time
}

// jsResource implements a resource.
type jsResource struct {
	Data  []string `json:"data"`
	Error string   `json:"error"`
}

const (
	jsModuleID module.ModuleID = "app.server.site.handler.js"

	jsResourceUnknown string = "unknown resource"

	jsConfigDefaultEnv            string = "production"
	jsConfigDefaultContainer      string = "root"
	jsConfigDefaultState          string = "state"
	jsConfigDefaultMaxVMs         int    = 4
	jsConfigDefaultVMTimeout      int    = 1000
	jsConfigDefaultVMHeapMaxBytes int    = 0
	jsConfigDefaultVMStackSize    int    = 0
	jsConfigDefaultCache          bool   = false
	jsConfigDefaultCacheTTL       int    = 60
	jsConfigDefaultCacheMaxItems  int    = 100
)

// jsOsOpen redirects to os.Open.
func jsOsOpen(name string) (*os.File, error) {
	return os.Open(name)
}

// jsOsOpenFile redirects to os.OpenFile.
func jsOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// jsOsReadFile redirects to os.ReadFile.
func jsOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// jsOsClose redirects to os.Close.
func jsOsClose(f *os.File) error {
	return f.Close()
}

// jsOsStat redirects to os.Stat.
func jsOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// jsJsonMarshal redirects to json.Marshal.
func jsJsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// init initializes the package.
func init() {
	module.Register(jsHandler{})
}

// ModuleInfo returns the module information.
func (h jsHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: jsModuleID,
		LoadModule: func() {
			gomonkey.Init()
		},
		UnloadModule: func() {
			gomonkey.ShutDown()
		},
		NewInstance: func() module.Module {
			return &jsHandler{
				logger:      slog.New(log.NewHandler(os.Stderr, string(jsModuleID), nil)),
				muIndex:     new(sync.RWMutex),
				muBundle:    new(sync.RWMutex),
				osOpen:      jsOsOpen,
				osOpenFile:  jsOsOpenFile,
				osReadFile:  jsOsReadFile,
				osClose:     jsOsClose,
				osStat:      jsOsStat,
				jsonMarshal: jsJsonMarshal,
			}
		},
	}
}

// Init initializes the handler.
func (h *jsHandler) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if h.config.Index == "" {
		h.logger.Error("Missing option or value", "option", "Index")
		errConfig = true
	} else {
		f, err := h.osOpenFile(h.config.Index, os.O_RDONLY, 0)
		if err != nil {
			h.logger.Error("Failed to open file", "option", "Index", "value", h.config.Index)
			errConfig = true
		} else {
			_ = h.osClose(f)
			fi, err := h.osStat(h.config.Index)
			if err != nil {
				h.logger.Error("Failed to stat file", "option", "Index", "value", h.config.Index)
				errConfig = true
			}
			if err == nil && fi.IsDir() {
				h.logger.Error("File is a directory", "option", "Index", "value", h.config.Index)
				errConfig = true
			}
		}
	}
	if h.config.Bundle == "" {
		h.logger.Error("Missing option or value", "option", "Bundle")
		errConfig = true
	} else {
		f, err := h.osOpenFile(h.config.Bundle, os.O_RDONLY, 0)
		if err != nil {
			h.logger.Error("Failed to open file", "option", "Bundle", "value", h.config.Bundle)
			errConfig = true
		} else {
			_ = h.osClose(f)
			fi, err := h.osStat(h.config.Bundle)
			if err != nil {
				h.logger.Error("Failed to stat file", "option", "Bundle", "value", h.config.Bundle)
				errConfig = true
			}
			if err == nil && fi.IsDir() {
				h.logger.Error("'File is a directory", "option", "Bundle", "value", h.config.Bundle)
				errConfig = true
			}
		}
	}
	if h.config.Env == nil {
		defaultValue := jsConfigDefaultEnv
		h.config.Env = &defaultValue
	}
	if *h.config.Env == "" {
		h.logger.Error("Invalid value", "option", "Env", "value", *h.config.Env)
		errConfig = true
	}
	if h.config.Container == nil {
		defaultValue := jsConfigDefaultContainer
		h.config.Container = &defaultValue
	}
	if *h.config.Container == "" {
		h.logger.Error("Invalid value", "option", "Container", "value", *h.config.Container)
		errConfig = true
	}
	if h.config.State == nil {
		defaultValue := jsConfigDefaultState
		h.config.State = &defaultValue
	}
	if *h.config.State == "" {
		h.logger.Error("Invalid value", "option", "State", "value", *h.config.State)
		errConfig = true
	}
	if h.config.MaxVMs == nil {
		defaultValue := jsConfigDefaultMaxVMs
		h.config.MaxVMs = &defaultValue
	}
	if *h.config.MaxVMs <= 0 {
		h.logger.Error("Invalid value", "option", "MaxVMs", "value", *h.config.MaxVMs)
		errConfig = true
	}
	if h.config.VMMaxHeapSize == nil {
		defaultValue := jsConfigDefaultVMHeapMaxBytes
		h.config.VMMaxHeapSize = &defaultValue
	}
	if *h.config.VMMaxHeapSize < 0 {
		h.logger.Error("Invalid value", "option", "VMHeapMaxBytes", "value", *h.config.VMMaxHeapSize)
		errConfig = true
	}
	if h.config.VMStackSize == nil {
		defaultValue := jsConfigDefaultVMStackSize
		h.config.VMStackSize = &defaultValue
	}
	if *h.config.VMStackSize < 0 {
		h.logger.Error("Invalid value", "option", "VMStackSize", "value", *h.config.VMStackSize)
		errConfig = true
	}
	if h.config.VMTimeout == nil {
		defaultValue := jsConfigDefaultVMTimeout
		h.config.VMTimeout = &defaultValue
	}
	if *h.config.VMTimeout <= 0 {
		h.logger.Error("Invalid value", "option", "VMTimeout", "value", *h.config.VMTimeout)
		errConfig = true
	}
	if h.config.Cache == nil {
		defaultValue := jsConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := jsConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}
	if *h.config.CacheTTL <= 0 {
		h.logger.Error("Invalid value", "option", "CacheTTL", "value", *h.config.CacheTTL)
		errConfig = true
	}
	if h.config.CacheMaxItems == nil {
		defaultValue := jsConfigDefaultCacheMaxItems
		h.config.CacheMaxItems = &defaultValue
	}
	if *h.config.CacheMaxItems <= 0 {
		h.logger.Error("Invalid value", "option", "CacheMaxCapacity", "value", *h.config.CacheMaxItems)
		errConfig = true
	}
	for index, rule := range h.config.Rules {
		if rule.Path == "" {
			h.logger.Error("Missing option or value", "rule", index+1, "option", "Path")
			errConfig = true
		} else {
			re, err := regexp.Compile(rule.Path)
			if err != nil {
				h.logger.Error("Invalid regular expression", "rule", index+1, "option", "Path", "value", rule.Path)
				errConfig = true
			} else {
				h.regexps = append(h.regexps, re)
			}
		}
		for _, state := range rule.State {
			if state.Key == "" {
				h.logger.Error("Missing option or value", "rule", index+1, "option", "Key")
				errConfig = true
			}
			if state.Resource == "" {
				h.logger.Error("Missing option or value", "rule", index+1, "option", "Resource")
				errConfig = true
			}
		}
	}

	if errConfig {
		return errors.New("config")
	}

	h.vms = make(chan struct{}, *h.config.MaxVMs)
	h.rwPool = render.NewRenderWriterPool()
	h.cache = newCache(*h.config.CacheMaxItems)

	return nil
}

// Register registers the handler.
func (h *jsHandler) Register(site core.ServerSite) error {
	h.site = site

	if err := site.RegisterHandler(h); err != nil {
		return fmt.Errorf("register handler: %v", err)
	}

	return nil
}

// Start starts the handler.
func (h *jsHandler) Start() error {
	if err := h.read(); err != nil {
		return fmt.Errorf("read: %v", err)
	}

	return nil
}

// Stop stops the handler.
func (h *jsHandler) Stop() error {
	h.muIndex.Lock()
	h.indexInfo = nil
	h.muIndex.Unlock()

	h.muBundle.Lock()
	h.bundleInfo = nil
	h.muBundle.Unlock()

	h.cache.Clear()

	return nil
}

// ServeHTTP implements the http handler.
func (h *jsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Path

	if *h.config.Cache {
		if item, ok := h.cache.Get(key).(*jsCacheItem); ok && item.expire.After(time.Now()) {
			render := item.render

			for key, values := range render.Header() {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			if render.Redirect() {
				http.Redirect(w, r, render.RedirectURL(), render.StatusCode())
				return
			}
			w.WriteHeader(render.StatusCode())
			if _, err := w.Write(render.Body()); err != nil {
				h.logger.Error("Failed to write render", "err", err)
				return
			}

			h.logger.Debug("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", true)

			return
		}
	}

	if err := h.read(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

		return
	}

	render, err := h.render(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

		return
	}

	if *h.config.Cache {
		h.cache.Set(key, &jsCacheItem{
			render: render,
			expire: time.Now().Add(time.Duration(*h.config.CacheTTL) * time.Second),
		})
	}

	for key, values := range render.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if render.Redirect() {
		http.Redirect(w, r, render.RedirectURL(), render.StatusCode())
		return
	}
	w.WriteHeader(render.StatusCode())
	if _, err := w.Write(render.Body()); err != nil {
		h.logger.Error("Failed to write render", "err", err)
		return
	}

	h.logger.Debug("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", false)
}

// read reads the application html and bundle files.
func (h *jsHandler) read() error {
	htmlInfo, err := h.osStat(h.config.Index)
	if err != nil {
		h.logger.Error("Failed to stat index file", "file", h.config.Index, "err", err)
		return fmt.Errorf("stat file %s: %v", h.config.Index, err)
	}

	h.muIndex.RLock()
	if h.indexInfo == nil || htmlInfo.ModTime().After(*h.indexInfo) {
		h.muIndex.RUnlock()

		buf, err := h.osReadFile(h.config.Index)
		if err != nil {
			h.logger.Error("Failed to read index file", "file", h.config.Index, "err", err)
			return fmt.Errorf("read file %s: %v", h.config.Index, err)
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
		h.logger.Error("Failed to stat bundle file", "file", h.config.Bundle, "err", err)
		return fmt.Errorf("stat file %s: %v", h.config.Bundle, err)
	}

	h.muBundle.RLock()
	if h.bundleInfo == nil || bundleInfo.ModTime().After(*h.bundleInfo) {
		h.muBundle.RUnlock()

		buf, err := h.osReadFile(h.config.Bundle)
		if err != nil {
			h.logger.Error("Failed to read bundle file", "file", h.config.Bundle, "err", err)
			return fmt.Errorf("read file %s: %v", h.config.Bundle, err)
		}

		h.muBundle.Lock()
		h.bundle = buf
		i := bundleInfo.ModTime()
		h.bundleInfo = &i
		h.muBundle.Unlock()
	} else {
		h.muBundle.RUnlock()
	}

	return nil
}

// render makes a new render.
func (h *jsHandler) render(r *http.Request) (render.Render, error) {
	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	var valid bool = true
	var mServerState map[string]jsResource
	var serverState *[]byte
	var mClientState map[string]jsResource
	var clientState *[]byte
	var vmResult *vmResult

	for index, rule := range h.config.Rules {
		m := h.regexps[index].FindStringSubmatch(r.URL.Path)
		if m == nil {
			continue
		}

		params := make(map[string]string)
		params["url"] = r.URL.Path
		if len(m) > 1 {
			for i, value := range m {
				if i > 0 {
					params[strconv.Itoa(i)] = value
				}
			}
			for i, name := range h.regexps[index].SubexpNames() {
				if i != 0 && name != "" {
					params[name] = m[i]
				}
			}
		}

		for _, entry := range rule.State {
			if mServerState == nil {
				mServerState = make(map[string]jsResource)
			}
			if mClientState == nil && entry.Export != nil && *entry.Export {
				mClientState = make(map[string]jsResource)
			}

			stateKey := h.replaceIndexRouteParameters(entry.Key, params)
			resourceKey := h.replaceIndexRouteParameters(entry.Resource, params)

			var resourceResult jsResource
			resource, err := h.site.Store().LoadResource(resourceKey)
			if err != nil {
				resourceResult.Error = jsResourceUnknown
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
				return nil, fmt.Errorf("marshal server state: %v", err)
			}

			serverState = &buf
		}

		if mClientState != nil {
			buf, err := h.jsonMarshal(mClientState)
			if err != nil {
				return nil, fmt.Errorf("marshal client state: %v", err)
			}

			clientState = &buf
		}

		if h.config.Rules[index].Last {
			break
		}
	}

	h.vms <- struct{}{}
	defer func() {
		<-h.vms
	}()
	vm, err := newVM(
		WithHeapMaxBytes(uint(*h.config.VMMaxHeapSize)),
		WithStackSize(uint(*h.config.VMStackSize)),
	)
	if err != nil {
		h.logger.Debug("Failed to create VM", "err", err)
		return nil, fmt.Errorf("create VM: %v", err)
	}

	h.muBundle.RLock()
	vmResult, err = vm.Execute(vmConfig{
		Env:     *h.config.Env,
		State:   serverState,
		Request: r,
		Site:    h.site,
	}, h.config.Bundle, h.bundle, time.Duration(*h.config.VMTimeout)*time.Millisecond)
	h.muBundle.RUnlock()
	if err != nil {
		h.logger.Debug("Failed to execute VM", "err", err)
		return nil, fmt.Errorf("execute VM: %v", err)
	}

	if vmResult.Redirect != nil && *vmResult.Redirect && vmResult.RedirectURL != nil && vmResult.RedirectStatus != nil {
		rw.WriteRedirect(*vmResult.RedirectURL, *vmResult.RedirectStatus)

		return rw.Render(), nil
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	for key, value := range vmResult.Headers {
		for _, v := range value {
			rw.Header().Add(key, v)
		}
	}
	if valid {
		rw.WriteHeader(*vmResult.Status)
	} else {
		rw.WriteHeader(http.StatusServiceUnavailable)
	}

	h.muIndex.RLock()
	if h.index != nil {
		err = h.doc(rw, r, bytes.NewReader(h.index), clientState, vmResult)
	} else {
		err = errors.New("index not loaded")
	}
	h.muIndex.RUnlock()
	if err != nil {
		h.logger.Debug("Failed to process render", "err", err)
		return nil, fmt.Errorf("process render: %v", err)
	}

	return rw.Render(), nil
}

// doc writes the final index.
func (h *jsHandler) doc(w render.RenderWriter, _ *http.Request, b io.Reader, state *[]byte, result *vmResult) error {
	doc, err := html.Parse(b)
	if err != nil {
		return fmt.Errorf("parse html: %v", err)
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
						Data: string(*state),
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

	if err := html.Render(w, doc); err != nil {
		return fmt.Errorf("render html: %v", err)
	}

	return nil
}

// replaceIndexRouteParameters returns a copy of the string s with all its parameters replaced.
func (h *jsHandler) replaceIndexRouteParameters(s string, params map[string]string) string {
	tmp := s
	for key, value := range params {
		tmp = strings.ReplaceAll(tmp, "$"+key, value)
	}
	return tmp
}

var _ core.ServerSiteMiddlewareModule = (*jsHandler)(nil)
