package file

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/bhuisgen/neon/pkg/core"
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
	Path       string `mapstructure:"path"`
	StatusCode *int   `mapstructure:"statusCode"`
	Cache      *bool  `mapstructure:"cache"`
	CacheTTL   *int   `mapstructure:"cacheTTL"`
}

// fileHandlerCache implments the file handler cache.
type fileHandlerCache struct {
	render render.Render
	expire time.Time
}

const (
	fileModuleID module.ModuleID = "server.site.handler.file"

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

// init initializes the module.
func init() {
	module.Register(fileHandler{})
}

// ModuleInfo returns the module information.
func (h fileHandler) ModuleInfo() module.ModuleInfo {
	return module.ModuleInfo{
		ID: fileModuleID,
		NewInstance: func() module.Module {
			return &fileHandler{
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
func (h *fileHandler) Init(config map[string]interface{}, logger *slog.Logger) error {
	h.logger = logger

	if err := mapstructure.Decode(config, &h.config); err != nil {
		h.logger.Error("Failed to parse configuration")
		return err
	}

	var errInit bool

	if h.config.Path == "" {
		h.logger.Error("Missing option or value", "option", "Path")
		errInit = true
	} else {
		f, err := h.osOpenFile(h.config.Path, os.O_RDONLY, 0)
		if err != nil {
			h.logger.Error("Failed to open file", "option", "Path", "value", h.config.Path)
			errInit = true
		} else {
			_ = h.osClose(f)
			fi, err := h.osStat(h.config.Path)
			if err != nil {
				h.logger.Error("Failed to stat file", "option", "Path", "value", h.config.Path)
				errInit = true
			}
			if err == nil && fi.IsDir() {
				h.logger.Error("File is a directory", "option", "Path", "value", h.config.Path)
				errInit = true
			}
		}
	}
	if h.config.StatusCode == nil {
		defaultValue := fileConfigDefaultStatusCode
		h.config.StatusCode = &defaultValue
	}
	if *h.config.StatusCode < 100 || *h.config.StatusCode > 599 {
		h.logger.Error("Invalid value", "option", "StatusCode", "value", *h.config.StatusCode)
		errInit = true
	}
	if h.config.Cache == nil {
		defaultValue := fileConfigDefaultCache
		h.config.Cache = &defaultValue
	}
	if h.config.CacheTTL == nil {
		defaultValue := fileConfigDefaultCacheTTL
		h.config.CacheTTL = &defaultValue
	}
	if *h.config.CacheTTL < 0 {
		h.logger.Error("Invalid value", "option", "CacheTTL", "value", *h.config.CacheTTL)
	}

	if errInit {
		return errors.New("init error")
	}

	h.rwPool = render.NewRenderWriterPool()

	return nil
}

// Register registers the handler.
func (h *fileHandler) Register(site core.ServerSite) error {
	err := site.RegisterHandler(h)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the handler.
func (h *fileHandler) Start() error {
	err := h.read()
	if err != nil {
		return err
	}

	return nil
}

// Stop stops the handler.
func (h *fileHandler) Stop() {
	h.muFile.Lock()
	h.file = nil
	h.fileInfo = nil
	h.muFile.Unlock()

	h.muCache.Lock()
	h.cache = nil
	h.muCache.Unlock()
}

// ServeHTTP implements the http handler.
func (h *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if *h.config.Cache {
		h.muCache.RLock()
		if h.cache != nil && h.cache.expire.After(time.Now()) {
			render := h.cache.render
			h.muCache.RUnlock()

			w.WriteHeader(render.StatusCode())
			if _, err := w.Write(render.Body()); err != nil {
				h.logger.Error("Failed to write render", "err", err)
				return
			}

			h.logger.Info("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", true)

			return
		}
		h.muCache.RUnlock()
	}

	err := h.read()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

		return
	}

	rw := h.rwPool.Get()
	defer h.rwPool.Put(rw)

	err = h.render(rw, r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		h.logger.Error("Render error", "url", r.URL.Path, "status", http.StatusServiceUnavailable)

		return
	}

	render := rw.Render()

	if *h.config.Cache {
		h.muCache.Lock()
		h.cache = &fileHandlerCache{
			render: render,
			expire: time.Now().Add(time.Duration(*h.config.CacheTTL) * time.Second),
		}
		h.muCache.Unlock()
	}

	w.WriteHeader(render.StatusCode())
	if _, err := w.Write(render.Body()); err != nil {
		h.logger.Error("Failed to write render", "err", err)
		return
	}

	h.logger.Info("Render completed", "url", r.URL.Path, "status", render.StatusCode(), "cache", false)
}

// read reads the file.
func (h *fileHandler) read() error {
	fileInfo, err := h.osStat(h.config.Path)
	if err != nil {
		h.logger.Debug("Failed to stat file", "file", h.config.Path, "err", err)

		return err
	}

	h.muFile.RLock()
	if h.fileInfo == nil || fileInfo.ModTime().After(*h.fileInfo) {
		h.muFile.RUnlock()
		buf, err := h.osReadFile(h.config.Path)
		if err != nil {
			h.logger.Debug("Failed to read file", "file", h.config.Path, "err", err)

			return err
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
func (h *fileHandler) render(w render.RenderWriter, r *http.Request) error {
	w.WriteHeader(*h.config.StatusCode)

	h.muFile.RLock()
	var err error
	if h.file != nil {
		_, err = w.Write(h.file)
	} else {
		err = errors.New("file not loaded")
	}
	h.muFile.RUnlock()
	if err != nil {
		return err
	}

	return nil
}

var _ core.ServerSiteHandlerModule = (*fileHandler)(nil)
