package app

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
	"runtime"
	"strconv"
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
	logger      *slog.Logger
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
	Index         string    `mapstructure:"index"`
	Bundle        string    `mapstructure:"bundle"`
	Env           *string   `mapstructure:"env"`
	Container     *string   `mapstructure:"container"`
	State         *string   `mapstructure:"state"`
	Timeout       *int      `mapstructure:"timeout"`
	MaxVMs        *int      `mapstructure:"maxVMs"`
	Cache         *bool     `mapstructure:"cache"`
	CacheTTL      *int      `mapstructure:"cacheTTL"`
	CacheMaxItems *int      `mapstructure:"cacheMaxItems"`
	Rules         []AppRule `mapstructure:"rules"`
}

// AppRule implements a rule.
type AppRule struct {
	Path  string              `mapstructure:"path"`
	State []AppRuleStateEntry `mapstructure:"state"`
	Last  bool                `mapstructure:"last"`
}

// AppRuleStateEntry implements a rule state entry.
type AppRuleStateEntry struct {
	Key      string `mapstructure:"key"`
	Resource string `mapstructure:"resource"`
	Export   *bool  `mapstructure:"export"`
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
	appConfigDefaultTimeout         int    = 200
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
func (h *appHandler) Init(config map[string]interface{}, logger *slog.Logger) error {
	h.logger = logger

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
		defaultValue := appConfigDefaultEnv
		h.config.Env = &defaultValue
	}
	if *h.config.Env == "" {
		h.logger.Error("Invalid value", "option", "Env", "name", *h.config.Env)
		errConfig = true
	}
	if h.config.Container == nil {
		defaultValue := appConfigDefaultContainer
		h.config.Container = &defaultValue
	}
	if *h.config.Container == "" {
		h.logger.Error("Invalid value", "option", "Container", "name", *h.config.Container)
		errConfig = true
	}
	if h.config.State == nil {
		defaultValue := appConfigDefaultState
		h.config.State = &defaultValue
	}
	if *h.config.State == "" {
		h.logger.Error("Invalid value", "option", "State", "name", *h.config.State)
		errConfig = true
	}
	if h.config.Timeout == nil {
		defaultValue := appConfigDefaultTimeout
		h.config.Timeout = &defaultValue
	}
	if *h.config.Timeout < 0 {
		h.logger.Error("Invalid value", "option", "Timeout", "name", *h.config.Timeout)
		errConfig = true
	}
	if h.config.MaxVMs == nil {
		defaultValue := runtime.GOMAXPROCS(0)
		h.config.MaxVMs = &defaultValue
	}
	if *h.config.MaxVMs < 0 {
		h.logger.Error("Invalid value", "option", "MaxVMs", "name", *h.config.MaxVMs)
		errConfig = true
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
		h.logger.Error("Invalid value", "option", "CacheTTL", "name", *h.config.CacheTTL)
		errConfig = true
	}
	if h.config.CacheMaxItems == nil {
		defaultValue := appConfigDefaultCacheMaxItems
		h.config.CacheMaxItems = &defaultValue
	}
	if *h.config.CacheMaxItems < 0 {
		h.logger.Error("Invalid value", "option", "CacheMaxCapacity", "name", *h.config.CacheMaxItems)
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
		return fmt.Errorf("register handler: %v", err)
	}

	return nil
}

// Start starts the handler.
func (h *appHandler) Start() error {
	err := h.read()
	if err != nil {
		return fmt.Errorf("read: %v", err)
	}

	return nil
}

// Stop stops the handler.
func (h *appHandler) Stop() error {
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
func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path

	if *h.config.Cache {
		if item, ok := h.cache.Get(key).(*appCacheItem); ok && item.expire.After(time.Now()) {
			render := item.render

			if render.Redirect() {
				http.Redirect(w, r, render.RedirectURL(), render.StatusCode())

				h.logger.Info("Render completed", "url", r.URL.Path, "redirect", render.RedirectURL(),
					"status", render.StatusCode(), "cache", true)
				return
			}

			for key, values := range render.Header() {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(render.StatusCode())
			if _, err := w.Write(render.Body()); err != nil {
				h.logger.Error("Failed to write render", "err", err)
				return
			}

			h.logger.Info("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", true)

			return
		}
	}

	err := h.read()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

		return
	}

	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	if err := h.render(rw, r); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

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

		h.logger.Error("Render completed", "url", r.URL.Path, "redirect", render.RedirectURL(),
			"status", http.StatusServiceUnavailable, "cache", false)

		return
	}

	for key, values := range rw.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(render.StatusCode())
	if _, err := w.Write(render.Body()); err != nil {
		h.logger.Error("Failed to write render", "err", err)
		return
	}

	h.logger.Info("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", false)
}

// read reads the application html and bundle files.
func (h *appHandler) read() error {
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
				mServerState = make(map[string]appResource)
			}
			if mClientState == nil && entry.Export != nil && *entry.Export {
				mClientState = make(map[string]appResource)
			}

			stateKey := h.replaceIndexRouteParameters(entry.Key, params)
			resourceKey := h.replaceIndexRouteParameters(entry.Resource, params)

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
				return fmt.Errorf("marshal server state: %v", err)
			}

			s := string(buf)
			serverState = &s
		}

		if mClientState != nil {
			buf, err := h.jsonMarshal(mClientState)
			if err != nil {
				return fmt.Errorf("marshal client state: %v", err)
			}

			s := string(buf)
			clientState = &s
		}

		if h.config.Rules[index].Last {
			break
		}
	}

	vm := h.vmPool.Get()
	defer h.vmPool.Put(vm)

	if err := vm.Configure(&vmConfig{
		Env:     *h.config.Env,
		Request: r,
		State:   serverState,
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)).With("site", h.site.Name()),
	); err != nil {
		h.logger.Debug("Failed to configure VM", "err", err)
		return fmt.Errorf("configure VM: %v", err)
	}

	var err error
	h.muBundle.RLock()
	result, err := vm.Execute(h.config.Bundle, h.bundle, time.Duration(*h.config.Timeout)*time.Second)
	h.muBundle.RUnlock()
	if err != nil {
		h.logger.Debug("Failed to execute VM", "err", err)
		return fmt.Errorf("execute VM: %v", err)
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
		h.logger.Debug("Failed to process render", "err", err)
		return fmt.Errorf("process render: %v", err)
	}

	return nil
}

// app writes the final index.
func (h *appHandler) app(w render.RenderWriter, r *http.Request, b io.Reader, state *string, result *vmResult) error {
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

	if err := html.Render(w, doc); err != nil {
		return fmt.Errorf("render html: %v", err)
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
