package file

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
	"github.com/bhuisgen/neon/pkg/log"
	"github.com/bhuisgen/neon/pkg/module"
	"github.com/bhuisgen/neon/pkg/render"
)

// fileHandler implements the file handler.
type fileHandler struct {
	config     *fileHandlerConfig
	logger     *slog.Logger
	file       []byte
	fileInfo   *time.Time
	muFile     *sync.RWMutex
	rwPool     render.RenderWriterPool
	cache      *fileHandlerCache
	muCache    *sync.RWMutex
	osOpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)
	osReadFile func(name string) ([]byte, error)
	osClose    func(*os.File) error
	osStat     func(name string) (fs.FileInfo, error)
}

// fileHandlerConfig implements the file handler configuration.
type fileHandlerConfig struct {
	Path        string  `mapstructure:"path"`
	ContentType *string `mapstructure:"contentType"`
	StatusCode  *int    `mapstructure:"statusCode"`
	Cache       *bool   `mapstructure:"cache"`
	CacheTTL    *int    `mapstructure:"cacheTTL"`
}

// fileHandlerCache implments the file handler cache.
type fileHandlerCache struct {
	render render.Render
	expire time.Time
}

const (
	fileModuleID module.ModuleID = "app.server.site.handler.file"

	fileConfigDefaultStatusCode int  = 200
	fileConfigDefaultCache      bool = false
	fileConfigDefaultCacheTTL   int  = 60
)

// fileOsOpenFile redirects to os.OpenFile.
func fileOsOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// fileOsReadFile redirects to os.ReadFile.
func fileOsReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// fileOsClose redirects to os.Close.
func fileOsClose(f *os.File) error {
	return f.Close()
}

// fileOsStat redirects to os.Stat.
func fileOsStat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// init initializes the package.
func init() {
	module.Register(fileHandler{})
}

// ModuleInfo returns the module information.
func (h fileHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID:           fileModuleID,
		LoadModule:   func() {},
		UnloadModule: func() {},
		NewInstance: func() module.Module {
			return &fileHandler{
				logger:     slog.New(log.NewHandler(os.Stderr, string(fileModuleID), nil)),
				muFile:     new(sync.RWMutex),
				muCache:    new(sync.RWMutex),
				osOpenFile: fileOsOpenFile,
				osReadFile: fileOsReadFile,
				osClose:    fileOsClose,
				osStat:     fileOsStat,
			}
		},
	}
}

// Init initializes the handler.
func (h *fileHandler) Init(config map[string]interface{}) error {
	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Error("Failed to parse configuration", "err", err)
		return fmt.Errorf("parse config: %v", err)
	}

	var errConfig bool

	if h.config.Path == "" {
		h.logger.Error("Missing option or value", "option", "Path")
		errConfig = true
	} else {
		f, err := h.osOpenFile(h.config.Path, os.O_RDONLY, 0)
		if err != nil {
			h.logger.Error("Failed to open file", "option", "Path", "value", h.config.Path)
			errConfig = true
		} else {
			_ = h.osClose(f)
			fi, err := h.osStat(h.config.Path)
			if err != nil {
				h.logger.Error("Failed to stat file", "option", "Path", "value", h.config.Path)
				errConfig = true
			}
			if err == nil && fi.IsDir() {
				h.logger.Error("File is a directory", "option", "Path", "value", h.config.Path)
				errConfig = true
			}
		}
	}
	if h.config.ContentType != nil && *h.config.ContentType == "" {
		h.logger.Error("Invalid value", "option", "ContentType")
		errConfig = true
	}
	if h.config.StatusCode == nil {
		defaultValue := fileConfigDefaultStatusCode
		h.config.StatusCode = &defaultValue
	}
	if *h.config.StatusCode < 100 || *h.config.StatusCode > 599 {
		h.logger.Error("Invalid value", "option", "StatusCode", "value", *h.config.StatusCode)
		errConfig = true
	}
	if h.config.Cache == nil {
		defaultValue := fileConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := fileConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}
	if *h.config.CacheTTL <= 0 {
		h.logger.Error("Invalid value", "option", "CacheTTL", "value", *h.config.CacheTTL)
	}

	if errConfig {
		return errors.New("config")
	}

	h.rwPool = render.NewRenderWriterPool()

	return nil
}

// Register registers the handler.
func (h *fileHandler) Register(site core.ServerSite) error {
	if err := site.RegisterHandler(h); err != nil {
		return fmt.Errorf("register handler: %v", err)
	}

	return nil
}

// Start starts the handler.
func (h *fileHandler) Start() error {
	if err := h.read(); err != nil {
		return fmt.Errorf("read: %v", err)
	}

	return nil
}

// Stop stops the handler.
func (h *fileHandler) Stop() error {
	h.muFile.Lock()
	h.file = nil
	h.fileInfo = nil
	h.muFile.Unlock()

	h.muCache.Lock()
	h.cache = nil
	h.muCache.Unlock()

	return nil
}

// ServeHTTP implements the http handler.
func (h *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if *h.config.Cache {
		h.muCache.RLock()
		if h.cache != nil && h.cache.expire.After(time.Now()) {
			render := h.cache.render
			h.muCache.RUnlock()

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

			h.logger.Debug("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", true)

			return
		}
		h.muCache.RUnlock()
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
		h.muCache.Lock()
		h.cache = &fileHandlerCache{
			render: render,
			expire: time.Now().Add(time.Duration(*h.config.CacheTTL) * time.Second),
		}
		h.muCache.Unlock()
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

	h.logger.Debug("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", false)
}

// read reads the file.
func (h *fileHandler) read() error {
	fileInfo, err := h.osStat(h.config.Path)
	if err != nil {
		h.logger.Error("Failed to stat file", "file", h.config.Path, "err", err)
		return fmt.Errorf("stat file %s: %v", h.config.Path, err)
	}

	h.muFile.RLock()
	if h.fileInfo == nil || fileInfo.ModTime().After(*h.fileInfo) {
		h.muFile.RUnlock()
		buf, err := h.osReadFile(h.config.Path)
		if err != nil {
			h.logger.Error("Failed to read file", "file", h.config.Path, "err", err)
			return fmt.Errorf("read file %s: %v", h.config.Path, err)
		}

		h.muFile.Lock()
		h.file = buf
		i := fileInfo.ModTime()
		h.fileInfo = &i
		h.muFile.Unlock()
	} else {
		h.muFile.RUnlock()
	}

	return nil
}

// render makes a new render.
func (h *fileHandler) render(_ *http.Request) (render.Render, error) {
	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	if h.config.ContentType != nil {
		rw.Header().Set("Content-Type", *h.config.ContentType)
	} else {
		contentType := mime.TypeByExtension(path.Ext(h.config.Path))
		if contentType != "" {
			rw.Header().Set("Content-Type", contentType)
		}
	}

	rw.WriteHeader(*h.config.StatusCode)

	h.muFile.RLock()
	var err error
	if h.file != nil {
		_, err = rw.Write(h.file)
	} else {
		err = errors.New("file not loaded")
	}
	h.muFile.RUnlock()
	if err != nil {
		h.logger.Error("Failed to process render", "err", err)
		return nil, fmt.Errorf("process render: %v", err)
	}

	return rw.Render(), nil
}

var _ core.ServerSiteHandlerModule = (*fileHandler)(nil)
